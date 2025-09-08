package cmd

import "github.com/takl/takl/sdk"

// sdkClient returns the default SDK client instance
func sdkClient() *sdk.Client {
	return sdk.DefaultClient()
}
