package main

import (
	"time"

	"github.com/GeertJohan/go.rice/embedded"
)

func init() {

	// define files
	file5 := &embedded.EmbeddedFile{
		Filename:    "api/auth/v1/login.html",
		FileModTime: time.Unix(1588786805, 0),

		Content: string("<!DOCTYPE html>\n<html>\n<head>\n    <title>Login</title>\n</head>\n<body>\n    <h1>Login</h1>\n    <hr>\n    <form method=\"POST\" action=\"/api/auth/v1/openid\">\n        <input type=\"submit\" value=\"Start OpenID Login\">\n    </form>\n    </body>\n</html>\n"),
	}

	// define dirs
	dir1 := &embedded.EmbeddedDir{
		Filename:   "",
		DirModTime: time.Unix(1588783770, 0),
		ChildFiles: []*embedded.EmbeddedFile{},
	}
	dir2 := &embedded.EmbeddedDir{
		Filename:   "api",
		DirModTime: time.Unix(1588783203, 0),
		ChildFiles: []*embedded.EmbeddedFile{},
	}
	dir3 := &embedded.EmbeddedDir{
		Filename:   "api/auth",
		DirModTime: time.Unix(1588783203, 0),
		ChildFiles: []*embedded.EmbeddedFile{},
	}
	dir4 := &embedded.EmbeddedDir{
		Filename:   "api/auth/v1",
		DirModTime: time.Unix(1588786310, 0),
		ChildFiles: []*embedded.EmbeddedFile{
			file5, // "api/auth/v1/login.html"

		},
	}

	// link ChildDirs
	dir1.ChildDirs = []*embedded.EmbeddedDir{
		dir2, // "api"

	}
	dir2.ChildDirs = []*embedded.EmbeddedDir{
		dir3, // "api/auth"

	}
	dir3.ChildDirs = []*embedded.EmbeddedDir{
		dir4, // "api/auth/v1"

	}
	dir4.ChildDirs = []*embedded.EmbeddedDir{}

	// register embeddedBox
	embedded.RegisterEmbeddedBox(`../../templates`, &embedded.EmbeddedBox{
		Name: `../../templates`,
		Time: time.Unix(1588783770, 0),
		Dirs: map[string]*embedded.EmbeddedDir{
			"":            dir1,
			"api":         dir2,
			"api/auth":    dir3,
			"api/auth/v1": dir4,
		},
		Files: map[string]*embedded.EmbeddedFile{
			"api/auth/v1/login.html": file5,
		},
	})
}
