package helper


// Get version of the project
// The file containing the version is created during Docker image creation and is not available with direct sources
func GetVersion() string {
	// The version is stored in file _VERSION
	version := GetFileContent("_VERSION")
	if version == "" {
		version = "n/a"
	}

	return version
}
