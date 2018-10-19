package v7action

//go:generate counterfeiter . UAAClient

type UAAClient interface {
	GetSSHPasscode(accessToken string, sshOAuthClient string) (string, error)
}
