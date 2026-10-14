package handlers

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin/binding"
)

func TestCreatePublicationRequestValidation(t *testing.T) {
	cases := []struct {
		name  string
		body  string
		valid bool
	}{
		{"valid", `{"title":"Mazda 3","description":"Unico dueno","price":45000000,"year":2019}`, true},
		{"missing title", `{"description":"x","price":1,"year":2019}`, false},
		{"missing description", `{"title":"x","price":1,"year":2019}`, false},
		{"missing price", `{"title":"x","description":"x","year":2019}`, false},
		{"zero price", `{"title":"x","description":"x","price":0,"year":2019}`, false},
		{"negative price", `{"title":"x","description":"x","price":-5,"year":2019}`, false},
		{"year 1900", `{"title":"x","description":"x","price":1,"year":1900}`, false},
		{"price as string", `{"title":"x","description":"x","price":"caro","year":2019}`, false},
		{"decimal year", `{"title":"x","description":"x","price":1,"year":2019.5}`, false},
	}

	for _, tc := range cases {
		var req CreatePublicationRequest
		err := binding.JSON.BindBody([]byte(strings.TrimSpace(tc.body)), &req)
		if (err == nil) != tc.valid {
			t.Errorf("%s: valid=%v, got err=%v", tc.name, tc.valid, err)
		}
	}
}
