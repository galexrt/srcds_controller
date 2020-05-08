package main

import (
	"time"

	"github.com/GeertJohan/go.rice/embedded"
)

func init() {

	// define files
	file3 := &embedded.EmbeddedFile{
		Filename:    "_partials/baseof.html",
		FileModTime: time.Unix(1588836463, 0),

		Content: string("{{ define \"baseof\" }}\n<!DOCTYPE html>\n<html>\n{{ template \"head\" . }}\n<body>\n    {{ template \"main\" . }}\n</body>\n</html>\n{{ end }}\n"),
	}
	file4 := &embedded.EmbeddedFile{
		Filename:    "_partials/head.html",
		FileModTime: time.Unix(1588835726, 0),

		Content: string("{{ define \"head\" }}\n    <head>\n        <!-- TODO -->\n        <title>{{ .Title }}</title>\n    </head>\n{{ end }}"),
	}
	file8 := &embedded.EmbeddedFile{
		Filename:    "api/auth/v1/login.html",
		FileModTime: time.Unix(1588836407, 0),

		Content: string("{{ template \"baseof\" . }}\n\n{{ define \"main\" }}\n<h1>Login</h1>\n<hr>\n<form method=\"POST\" action=\"/api/auth/v1/openid\">\n    <input type=\"submit\" value=\"Start OpenID Login\">\n</form>\n{{ end }}\n"),
	}

	// define dirs
	dir1 := &embedded.EmbeddedDir{
		Filename:   "",
		DirModTime: time.Unix(1588836436, 0),
		ChildFiles: []*embedded.EmbeddedFile{},
	}
	dir2 := &embedded.EmbeddedDir{
		Filename:   "_partials",
		DirModTime: time.Unix(1588836436, 0),
		ChildFiles: []*embedded.EmbeddedFile{
			file3, // "_partials/baseof.html"
			file4, // "_partials/head.html"

		},
	}
	dir5 := &embedded.EmbeddedDir{
		Filename:   "api",
		DirModTime: time.Unix(1588783203, 0),
		ChildFiles: []*embedded.EmbeddedFile{},
	}
	dir6 := &embedded.EmbeddedDir{
		Filename:   "api/auth",
		DirModTime: time.Unix(1588783203, 0),
		ChildFiles: []*embedded.EmbeddedFile{},
	}
	dir7 := &embedded.EmbeddedDir{
		Filename:   "api/auth/v1",
		DirModTime: time.Unix(1588786310, 0),
		ChildFiles: []*embedded.EmbeddedFile{
			file8, // "api/auth/v1/login.html"

		},
	}

	// link ChildDirs
	dir1.ChildDirs = []*embedded.EmbeddedDir{
		dir2, // "_partials"
		dir5, // "api"

	}
	dir2.ChildDirs = []*embedded.EmbeddedDir{}
	dir5.ChildDirs = []*embedded.EmbeddedDir{
		dir6, // "api/auth"

	}
	dir6.ChildDirs = []*embedded.EmbeddedDir{
		dir7, // "api/auth/v1"

	}
	dir7.ChildDirs = []*embedded.EmbeddedDir{}

	// register embeddedBox
	embedded.RegisterEmbeddedBox(`../../templates`, &embedded.EmbeddedBox{
		Name: `../../templates`,
		Time: time.Unix(1588836436, 0),
		Dirs: map[string]*embedded.EmbeddedDir{
			"":            dir1,
			"_partials":   dir2,
			"api":         dir5,
			"api/auth":    dir6,
			"api/auth/v1": dir7,
		},
		Files: map[string]*embedded.EmbeddedFile{
			"_partials/baseof.html":  file3,
			"_partials/head.html":    file4,
			"api/auth/v1/login.html": file8,
		},
	})
}
