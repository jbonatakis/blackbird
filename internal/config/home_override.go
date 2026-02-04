package config

// SetUserHomeDirForTest overrides the home directory resolver.
// It returns a restore function to reset the original resolver.
func SetUserHomeDirForTest(fn func() (string, error)) func() {
	orig := userHomeDir
	userHomeDir = fn
	return func() {
		userHomeDir = orig
	}
}
