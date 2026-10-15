package cli

import "github.com/zalando/go-keyring"

func init() { keyring.MockInit() }
