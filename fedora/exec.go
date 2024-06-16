package main

// WithExecScripts adds scripts (Files) to be uploaded and executed on the
// generated Container image
func (f *Fedora) WithExecScripts(
	// scripts (Files) to be uploaded and executed
	scripts []*File,
	// if true, the script(s) will be run prior to any packages being installed
	// on the Container image
	// if false, they will be run after packages are installed as part of
	// WithPackagesInstalled
	prePackages bool,
) *Fedora {
	if prePackages {
		f.ExecScriptPre = append(f.ExecScriptPre, scripts...)
	} else {
		f.ExecScriptPost = append(f.ExecScriptPost, scripts...)
	}

	return f
}

// WithExec will execute the given command on the Container image
func (f *Fedora) WithExec(
	// the command to be executed
	command []string,
	// if true, the command will be run prior to any packages being installed
	// on the Container image
	// if false, it will be run after packages are installed as part of
	// WithPackagesInstalled
	prePackages bool,
) *Fedora {
	if prePackages {
		f.ExecPre = append(f.ExecPre, command)
	} else {
		f.ExecPost = append(f.ExecPost, command)
	}

	return f
}
