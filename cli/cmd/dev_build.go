//go:build dev
// +build dev

package cmd

func init() {
	// Add dev command only in dev builds
	rootCmd.AddCommand(devCmd)
}