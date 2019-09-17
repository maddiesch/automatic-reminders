package auto

import "net/url"

const (
	AutomaticIndexSortKeyValue = "_AUTOMATIC_ACCOUNT"
)

func AutomaticAccountsURL(path string) *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   "accounts.automatic.com",
		Path:   path,
	}
}

func AutomaticAPIURL(path string) *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   "api.automatic.com",
		Path:   path,
	}
}
