package fuzznexus

import "testing"

func TestHostURLFromRequest(t *testing.T) {
	tests := []struct {
		name string
		req  Request
		want string
	}{
		{
			name: "trims final path component",
			req:  Request{Url: "https://example.test/a/b", Host: "example.test"},
			want: "example.test/a",
		},
		{
			name: "invalid URL falls back to host",
			req:  Request{Url: "://bad-url", Host: "example.test"},
			want: "example.test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HostURLFromRequest(tt.req); got != tt.want {
				t.Fatalf("HostURLFromRequest() = %q, want %q", got, tt.want)
			}
		})
	}
}
