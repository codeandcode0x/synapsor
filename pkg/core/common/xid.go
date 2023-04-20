package common

import "github.com/rs/xid"

func GenXid() string {
	guid := xid.New()
	return guid.String()
}
