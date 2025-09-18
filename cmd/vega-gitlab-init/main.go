package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/cenkalti/backoff/v4"
	"github.com/fatih/color"
	"github.com/go-faster/errors"
	"github.com/xanzy/go-gitlab"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	apply "k8s.io/client-go/applyconfigurations/core/v1"
	applyMeta "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/go-faster/vega"
	"github.com/go-faster/vega/internal/cli"
	"github.com/go-faster/vega/internal/k8s"
)

type Client struct {
	http *http.Client
	base string
}

func Helm(ctx context.Context, args ...string) error {
	if p := os.Getenv("HELM_PROXY"); p != "" {
		_ = os.Setenv("HTTPS_PROXY", p)
	}

	install := func() error {
		cmd := exec.CommandContext(ctx, "helm", args...)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return errors.Wrap(err, "helm")
		}
		return nil
	}

	var retries int
	bo := backoff.WithContext(
		backoff.NewExponentialBackOff(),
		ctx,
	)

	return backoff.RetryNotify(install, bo, func(err error, d time.Duration) {
		retries++
		fmt.Println(
			color.New(color.FgYellow).Sprint("Retrying helm install:"),
			color.New(color.Bold, color.FgYellow).Sprint(retries),
		)
	})
}

func (c *Client) ProjectRunnerRegistrationToken(ctx context.Context, projectPath string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+projectPath+"/-/settings/ci_cd", http.NoBody)
	if err != nil {
		return "", errors.Wrap(err, "new request")
	}
	res, err := cli.Response(ctx, c.http, req)
	if err != nil {
		return "", errors.Wrap(err, "do")
	}
	defer func() { _ = res.Body.Close() }()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", errors.Wrap(err, "new document")
	}

	token := doc.Find("#registration_token").First().Text()
	if token == "" {
		return "", errors.New("authenticity_token not found")
	}

	return token, nil
}

func (c *Client) Token(ctx context.Context, ref string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+ref, http.NoBody)
	if err != nil {
		return "", errors.Wrap(err, "new request")
	}
	res, err := cli.Response(ctx, c.http, req)
	if err != nil {
		return "", errors.Wrap(err, "do")
	}
	defer func() { _ = res.Body.Close() }()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", errors.Wrap(err, "new document")
	}
	token, ok := doc.Find("input[name=authenticity_token]").First().Attr("value")
	if !ok {
		return "", errors.New("authenticity_token not found")
	}
	if token == "" {
		return "", errors.New("authenticity_token is empty")
	}

	return token, nil
}

type Auth struct {
	Username string
	Password string
	Token    string
}

type AccessToken struct {
	Name      string
	Scopes    []string
	ExpiresAt time.Time
}

func (c *Client) csrfToken(ctx context.Context, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, http.NoBody)
	if err != nil {
		return "", errors.Wrap(err, "new request")
	}
	res, err := cli.Response(ctx, c.http, req)
	if err != nil {
		return "", errors.Wrap(err, "do")
	}
	defer func() { _ = res.Body.Close() }()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", errors.Wrap(err, "new document")
	}
	token, ok := doc.Find("meta[name=csrf-token]").First().Attr("content")
	if !ok {
		return "", errors.New("csrf-token not found")
	}
	if token == "" {
		return "", errors.New("csrf-token is empty")
	}

	return token, nil
}

func (c *Client) CreateAccessToken(ctx context.Context, opt AccessToken) (string, error) {
	const ref = "/-/profile/personal_access_tokens"

	csrfToken, err := c.csrfToken(ctx, ref)
	if err != nil {
		return "", errors.Wrap(err, "get csrf token")
	}

	f := url.Values{}
	f.Set("commit", "Create personal access token")
	f.Set("personal_access_token[name]", opt.Name)
	f.Set("personal_access_token[expires_at]", opt.ExpiresAt.Format("2006-01-02"))
	for _, s := range opt.Scopes {
		f.Add("personal_access_token[scopes][]", s)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+ref, strings.NewReader(f.Encode()))
	if err != nil {
		return "", errors.Wrap(err, "new request")
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("X-CSRF-Token", csrfToken)
	res, err := cli.Response(ctx, c.http, req)
	if err != nil {
		return "", errors.Wrap(err, "do")
	}
	defer func() { _ = res.Body.Close() }()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", errors.Wrap(err, "read body")
	}

	var data struct {
		NewToken string `json:"new_token"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Println("body:", string(body))
		return "", errors.Wrap(err, "parse token")
	}
	if data.NewToken == "" {
		return "", errors.New("token not found")
	}

	return data.NewToken, nil
}

func (c *Client) Auth(ctx context.Context, auth Auth) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	f := url.Values{}
	f.Set("authenticity_token", auth.Token)
	f.Set("user[login]", auth.Username)
	f.Set("user[password]", auth.Password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/users/sign_in", strings.NewReader(f.Encode()))
	if err != nil {
		return errors.Wrap(err, "new request")
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if err := cli.Do(ctx, c.http, req); err != nil {
		return errors.Wrap(err, "do")
	}

	return nil
}

type Application struct {
	Name        string
	RedirectURI string
	Scopes      []string
}

type ApplicationCredentials struct {
	ID     string
	Secret string
}

func (c *Client) AddApplication(ctx context.Context, app Application) (*ApplicationCredentials, error) {
	token, err := c.Token(ctx, "/admin/applications/new")
	if err != nil {
		return nil, errors.Wrap(err, "get token")
	}
	f := url.Values{}
	f.Set("authenticity_token", token)
	f.Set("doorkeeper_application[name]", app.Name)
	f.Set("doorkeeper_application[redirect_uri]", app.RedirectURI)
	f.Set("doorkeeper_application[trusted]", "0")
	f.Set("doorkeeper_application[confidential]", "0")
	f.Set("doorkeeper_application[confidential]", "1")
	for _, s := range app.Scopes {
		f.Add("doorkeeper_application[scopes][]", s)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/admin/applications", strings.NewReader(f.Encode()))
	if err != nil {
		return nil, errors.Wrap(err, "new request")
	}
	req.Header.Set("Referer", c.base+"/admin/applications/new")
	req.Header.Set("Origin", c.base)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res, err := cli.Response(ctx, c.http, req)
	if err != nil {
		return nil, errors.Wrap(err, "do")
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "new document")
	}

	// Check for error.
	if v := strings.TrimSpace(doc.Find("#error_explanation").Find("h4").Text()); v != "" {
		return nil, errors.Errorf("%s", v)
	}
	// Parse credentials.
	creds := &ApplicationCredentials{
		ID:     doc.Find("#application_id").First().AttrOr("value", ""),
		Secret: doc.Find(`button[data-title="Copy secret"]`).First().AttrOr("data-clipboard-text", ""),
	}
	if creds.ID == "" {
		return nil, errors.New("id not found")
	}
	if creds.Secret == "" {
		return nil, errors.New("secret not found")
	}

	return creds, nil
}

func run(ctx context.Context) error {
	opts := k8s.OptionsFromFlags()
	base := flag.String("base", "", "base url (without trailing slash)")
	flag.Parse()

	if *base == "" {
		return errors.New("--base not specified")
	}

	config, err := opts.Config()
	if err != nil {
		return errors.Wrap(err, "k8s: build config")
	}
	kube, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "k8s: new")
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return errors.Wrap(err, "new")
	}
	client := &Client{
		base: *base,
		http: &http.Client{
			Jar:     jar,
			Timeout: time.Second * 15,
			Transport: &http.Transport{
				MaxIdleConns:          10,
				IdleConnTimeout:       time.Second * 30,
				ResponseHeaderTimeout: time.Second * 5,
			},
		},
	}

	// 1. Find CSRF (authenticity) token.
	token, err := client.Token(ctx, "/users/sign_in")
	if err != nil {
		return errors.Wrap(err, "token")
	}

	// 2. Sign in.
	if err := client.Auth(ctx, Auth{
		Username: k8s.GitLabDebugRootLogin,
		Password: k8s.GitLabDebugRootPassword,
		Token:    token,
	}); err != nil {
		return errors.Wrap(err, "auth")
	}

	// 3. Create API token.
	accessToken, err := client.CreateAccessToken(ctx, AccessToken{
		Name: "root",
		Scopes: []string{
			"api",
			"read_api",
			"read_user",
			"read_repository",
			"write_repository",
			"sudo",
		},
		ExpiresAt: time.Now().Add(time.Hour * 24 * 7),
	})
	if err != nil {
		return errors.Wrap(err, "create access token")
	}

	fmt.Println(
		color.New(color.FgGreen).Sprint("Created root access token:"),
		color.New(color.Bold, color.FgCyan).Sprint(accessToken),
	)

	// Save token as kubernetes secret.
	var (
		name         = "vega.gitlab"
		internalURL  = fmt.Sprintf("http://gitlab.vega.svc.cluster.local")
		appName      = "gitlab"
		annotations  = map[string]string{}
		applyOptions = meta.ApplyOptions{
			FieldManager: "gitlab-init",
		}
		typeConfig = applyMeta.TypeMetaApplyConfiguration{
			Kind:       k8s.String("Secret"),
			APIVersion: k8s.String("v1"),
		}
		labels = func(secretType string) map[string]string {
			return map[string]string{
				vega.LabelSecretType: secretType,

				vega.LabelApplication: appName,
				vega.LabelUnit:        k8s.Name,
				vega.LabelProject:     k8s.Name,

				k8s.LabelApp:       name,
				k8s.LabelName:      appName,
				k8s.LabelPartOf:    k8s.Name,
				k8s.LabelCreatedBy: applyOptions.FieldManager,
				k8s.LabelManagedBy: applyOptions.FieldManager,
			}
		}
	)
	kubeSecret, err := kube.CoreV1().Secrets(k8s.Namespace).Apply(ctx, &apply.SecretApplyConfiguration{
		TypeMetaApplyConfiguration: typeConfig,
		ObjectMetaApplyConfiguration: &applyMeta.ObjectMetaApplyConfiguration{
			Annotations: annotations,
			Labels:      labels(vega.SecretTypeGitLabAccessToken),
			Name:        k8s.String(name + ".access"),
		},
		Data: map[string][]byte{
			"token": []byte(accessToken),
		},
	}, applyOptions)
	if err != nil {
		return errors.Wrapf(err, "apply %q secret", name+".access")
	}

	fmt.Println(
		color.New(color.FgGreen).Sprint("Saved as kubernetes secret:"),
		color.New(color.Bold, color.FgCyan).Sprint(kubeSecret.Name),
	)

	// Use API with created token.
	gl, err := gitlab.NewClient(accessToken, gitlab.WithBaseURL(*base+"/api/v4"))
	if err != nil {
		return errors.Wrap(err, "create new gitlab client")
	}
	// Create gitlab project.
	project, _, err := gl.Projects.CreateProject(&gitlab.CreateProjectOptions{
		Name: k8s.String("test"),
	}, gitlab.WithContext(ctx))
	errRes, ok := errors.Into[*gitlab.ErrorResponse](err)
	switch {
	case ok && strings.Contains(errRes.Message, "has already been taken"):
		// Sometimes gitlab returns this error for some reason.
		// Non-idempotent retries?
		//
		// Just ignoring, should be fine.
		fmt.Println(
			color.New(color.FgYellow).Sprint("Project already exists"),
		)
	case err != nil:
		// Non-recoverable error.
		return errors.Wrap(err, "create project")
	default:
		fmt.Println(
			color.New(color.FgGreen).Sprint("Created GitLab project:"),
			color.New(color.Bold, color.FgCyan).Sprint(project.ID),
		)
	}

	// Save CI/CD runner registration token for test project.
	regToken, err := client.ProjectRunnerRegistrationToken(ctx, "/root/test")
	if err != nil {
		return errors.Wrap(err, "project runner registration token")
	}
	regSecret, err := kube.CoreV1().Secrets(k8s.Namespace).Apply(ctx, &apply.SecretApplyConfiguration{
		TypeMetaApplyConfiguration: typeConfig,
		ObjectMetaApplyConfiguration: &applyMeta.ObjectMetaApplyConfiguration{
			Annotations: annotations,
			Labels:      labels(vega.SecretTypeGitLabRunnerRegistrationToken),
			Name:        k8s.String(name + ".reg"),
		},
		Data: map[string][]byte{
			"token": []byte(regToken),
		},
	}, applyOptions)
	if err != nil {
		return errors.Wrap(err, "save registration token")
	}

	fmt.Println(
		color.New(color.FgGreen).Sprint("Saved runner registration token:"),
		color.New(color.Bold, color.FgCyan).Sprint(regSecret.Name),
	)

	appCreds, err := client.AddApplication(ctx, Application{
		Name:        "vega",
		RedirectURI: "http://api.localhost/auth/callback",
		Scopes: []string{
			"openid", "profile", "email",
		},
	})
	if err != nil {
		return errors.Wrap(err, "add application")
	}
	appSecret, err := kube.CoreV1().Secrets(k8s.Namespace).Apply(ctx, &apply.SecretApplyConfiguration{
		TypeMetaApplyConfiguration: typeConfig,
		ObjectMetaApplyConfiguration: &applyMeta.ObjectMetaApplyConfiguration{
			Annotations: annotations,
			Labels:      labels(vega.SecretTypeGitLabApplicationCredentials),
			Name:        k8s.String(name + ".app"),
		},
		Data: map[string][]byte{
			"id":     []byte(appCreds.ID),
			"secret": []byte(appCreds.Secret),
		},
	}, applyOptions)
	if err != nil {
		return errors.Wrapf(err, "apply %q secret", name+".app")
	}

	fmt.Println(
		color.New(color.FgGreen).Sprint("Saved OAuth2 application credentials:"),
		color.New(color.Bold, color.FgCyan).Sprint(appSecret.Name),
	)

	// Installing gitlab-runner with helm.
	if err := Helm(ctx,
		"upgrade",
		"--install",
		"--namespace", k8s.Namespace, // same ns to ease firewall
		"--repo", "https://charts.gitlab.io",
		"gitlab-runner", "gitlab-runner",
		"--set", fmt.Sprintf(`gitlabUrl=%s`, internalURL),
		"--set", fmt.Sprintf(`runnerRegistrationToken=%s`, regToken),
		"--values", "_hack/gitlab.yml",
	); err != nil {
		return errors.Wrap(err, "install gitlab-runner")
	}

	fmt.Println(
		color.New(color.Bold, color.FgGreen).Sprint("Installed gitlab runner"),
	)

	return nil
}

func main() {
	cli.Run(run)
}
