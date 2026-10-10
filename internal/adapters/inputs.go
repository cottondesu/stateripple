package adapters

func (p goPackage) nativeFiles() []string {
	files := []string{}
	for _, group := range [][]string{p.SFiles, p.CFiles, p.HFiles, p.CXXFiles, p.MFiles, p.FFiles, p.SwigFiles, p.SwigCXXFiles, p.SysoFiles} {
		files = append(files, group...)
	}
	return files
}

func (p goPackage) analysisFiles() []string {
	files := p.nativeFiles()
	for _, group := range [][]string{p.GoFiles, p.CgoFiles, p.TestGoFiles, p.XTestGoFiles, p.EmbedFiles, p.TestEmbedFiles, p.XTestEmbedFiles} {
		files = append(files, group...)
	}
	return files
}
