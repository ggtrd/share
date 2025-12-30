package helper


// Get version of the project
// The file containing the version is created during Docker image creation and is not available with direct sources
func GetVersion(fileVersion string) string {
	version := GetFileContent(fileVersion)
	if version == "" {
		version = "n/a"
	}

	return version
}
