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
	path := "static/dynamic"

	var dependencies = []StaticDependency{
		StaticDependency{"https://unpkg.com/openpgp@latest/dist/openpgp.min.js", "static/dynamic/openpgp.min.js"},
	}

	CreatePath(path)

	for _, dependency := range dependencies {
		if ! FileExists(dependency.localPath) {
			err := DownloadFile(dependency.url, dependency.localPath)
			if err != nil {
				log.Println("error:", err)
			}
		}
	}
}