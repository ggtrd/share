package helper

import (
	"log"
)


type StaticDependency struct {
	url string
	localPath string
}


// Download static dependencies (like Javascript libraries etc...)
func DownloadStaticDependencies() {
	// General path for most of the files
	CreatePath("static/dynamic")


	// Specific path for Font-Awesome
	CreatePath("static/dynamic/fonts")
	CreatePath("static/dynamic/fonts/css")
	CreatePath("static/dynamic/fonts/webfonts")


	var dependencies = []StaticDependency{
		StaticDependency{"https://unpkg.com/openpgp@latest/dist/openpgp.min.js", "static/dynamic/openpgp.min.js"},
		StaticDependency{"https://unpkg.com/slate@latest/dist/index.js", "static/dynamic/slate.js"},
		StaticDependency{"https://raw.githubusercontent.com/FortAwesome/Font-Awesome/refs/tags/7.1.0/css/all.min.css", "static/dynamic/fonts/css/fontawesome.all.min.css"},
		StaticDependency{"https://github.com/FortAwesome/Font-Awesome/raw/refs/tags/7.1.0/webfonts/fa-brands-400.woff2", "static/dynamic/fonts/webfonts/fa-brands-400.woff2"},
		StaticDependency{"https://github.com/FortAwesome/Font-Awesome/raw/refs/tags/7.1.0/webfonts/fa-solid-900.woff2", "static/dynamic/fonts/webfonts/fa-solid-900.woff2"},
	}


	for _, dependency := range dependencies {
		if ! FileExists(dependency.localPath) {
			err := DownloadFile(dependency.url, dependency.localPath)
			if err != nil {
				log.Println("error:", err)
			}
		}
	}
}