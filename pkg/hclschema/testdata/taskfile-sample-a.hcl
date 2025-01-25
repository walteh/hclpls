version = "3"

tasks {
	generate-taskfiles {
		cmds = [
			"./scripts/setup-tools-for-local.sh --generate-taskfiles --skip-build",
		]
	}
}
