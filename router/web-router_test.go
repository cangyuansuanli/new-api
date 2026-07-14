package router

import "testing"

func TestShouldReturnNotFound(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "api route", path: "/api/token/", want: true},
		{name: "relay route", path: "/v1/models", want: true},
		{name: "asset route", path: "/assets/missing.js", want: true},
		{name: "default frontend static route", path: "/static/js/async/missing.js", want: true},
		{name: "frontend page", path: "/keys", want: false},
		{name: "nested frontend page", path: "/dashboard/overview", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldReturnNotFound(test.path); got != test.want {
				t.Fatalf("shouldReturnNotFound(%q) = %v, want %v", test.path, got, test.want)
			}
		})
	}
}
