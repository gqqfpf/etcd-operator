package file

import "net/url"

func ParseBackupUrl(backupurl string) (string, string, string, error) {
	parse, err := url.Parse(backupurl)
	if err != nil {
		return "", "", "", err
	}
	return parse.Scheme, parse.Host, parse.Path[1:], err
}
