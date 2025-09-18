package k8s

func Bool(v bool) *bool { return &v }
func True() *bool       { return Bool(true) }
func False() *bool      { return Bool(false) }

func Int(v int) *int32 {
	t := int32(v)
	return &t
}

func Int64(v int64) *int64 {
	return &v
}

func String(v string) *string {
	return &v
}

func T[T any](v T) *T {
	return &v
}
