package installer

// Construct maps configs to results.
func Construct[Config, Result any](f func(Config) Result, configs ...Config) (res []Result) {
	for _, config := range configs {
		res = append(res, f(config))
	}
	return res
}
