// +build dev

package swagger

import "net/http"

var Browser = func() http.FileSystem {
	return http.Dir("swagger/static")
}()
