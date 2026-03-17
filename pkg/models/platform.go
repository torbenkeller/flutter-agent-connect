package models

type PlatformType string

const (
	PlatformIOS     PlatformType = "ios"
	PlatformAndroid PlatformType = "android"
	PlatformWeb     PlatformType = "web"
	PlatformMacOS   PlatformType = "macos"
)

func (p PlatformType) IsValid() bool {
	switch p {
	case PlatformIOS, PlatformAndroid, PlatformWeb, PlatformMacOS:
		return true
	}
	return false
}
