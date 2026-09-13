package main

import (
	"runtime"
	"strings"
	"sync"
)

// buildReport gathers everything the renderer needs. Per-module git calls run
// concurrently because they are independent and dominate the runtime.
func buildReport(path string, o Options) (Report, error) {
	root, err := resolveRoot(path)
	if err != nil {
		return Report{}, err
	}

	rep := Report{Window: o.Window}
	if o.Fetch {
		if err := fetch(root); err != nil {
			rep.Warnings = append(rep.Warnings, "fetch failed, reporting on local refs: "+firstLine(err.Error()))
		}
	}

	ref, err := pickRef(root, o.Branch)
	if err != nil {
		return Report{}, err
	}
	rep.Ref = ref

	base, hasGitHub := remoteBase(root)
	if hasGitHub {
		rep.Repo = strings.TrimPrefix(base, "https://github.com/")
	}

	tree, err := listTree(root, ref.Rev)
	if err != nil {
		return Report{}, err
	}
	mods := discoverModules(tree)
	rep.RootReadme = rootReadme(root, ref, base, hasGitHub, mods, tree, o)

	reports := make([]ModuleReport, len(mods))
	errs := make([]error, len(mods))
	sem := make(chan struct{}, max(4, runtime.NumCPU()))
	var wg sync.WaitGroup

	for i, m := range mods {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			reports[i], errs[i] = collect(root, ref, base, hasGitHub, m, o)
		}()
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return Report{}, err
		}
	}

	sortReports(reports)
	rep.Modules = reports
	return rep, nil
}

// rootReadme surfaces the repository's own README when the root is not itself
// a module, so a docs-shaped repo still leads with its overview.
func rootReadme(root string, ref Ref, base string, hasGitHub bool, mods []Module, tree []string, o Options) Readme {
	for _, m := range mods {
		if m.Dir == "." {
			return Readme{}
		}
	}

	path := ""
	for _, p := range tree {
		if strings.EqualFold(p, "README.md") {
			path = p
			break
		}
	}
	if path == "" {
		return Readme{}
	}

	lines, truncated, total, err := readLines(root, ref.Rev, path, o.Lines)
	if err != nil {
		return Readme{}
	}
	r := Readme{Lines: lines, Truncated: truncated, Total: total, URL: path}
	if hasGitHub {
		r.URL = blobURL(base, ref.Name, path)
	}
	return r
}

func collect(root string, ref Ref, base string, hasGitHub bool, m Module, o Options) (ModuleReport, error) {
	mr := ModuleReport{Module: m}

	churn, err := moduleChurn(root, ref.Rev, m, o.Window)
	if err != nil {
		return mr, err
	}
	mr.Churn = churn

	if m.Readme != "" {
		lines, truncated, total, err := readLines(root, ref.Rev, m.Readme, o.Lines)
		if err != nil {
			return mr, err
		}
		mr.ReadmeLines, mr.Truncated, mr.ReadmeTotal = lines, truncated, total
		if hasGitHub {
			mr.URL = blobURL(base, ref.Name, m.Readme)
		} else {
			mr.URL = m.Readme
		}
	}

	if o.ShowCommit {
		last, err := lastCommit(root, ref.Rev, m)
		if err != nil {
			return mr, err
		}
		mr.Last = last
	}
	return mr, nil
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}
