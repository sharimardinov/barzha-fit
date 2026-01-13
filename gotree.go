package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type FuncInfo struct {
	Name      string
	Recv      string // receiver type (for methods)
	Signature string
}

type FileInfo struct {
	Path  string
	Funcs []FuncInfo
	Types map[string][]FuncInfo // type -> methods
	Pkg   string
}

func main() {
	root := flag.String("root", ".", "project root")
	maxDepth := flag.Int("depth", 8, "max directory depth")
	showPrivate := flag.Bool("private", true, "include unexported (lowercase) identifiers")
	flag.Parse()

	absRoot, err := filepath.Abs(*root)
	must(err)

	files, err := collectGoFiles(absRoot, *maxDepth)
	must(err)

	infos := make([]FileInfo, 0, len(files))
	for _, p := range files {
		fi, ok := parseGoFile(p)
		if !ok {
			continue
		}
		// фильтр приватных
		if !*showPrivate {
			fi.Funcs = filterExportedFuncs(fi.Funcs)
			for t, ms := range fi.Types {
				fi.Types[t] = filterExportedFuncs(ms)
			}
		}
		infos = append(infos, fi)
	}

	printTree(absRoot, infos)
}

func collectGoFiles(root string, maxDepth int) ([]string, error) {
	var out []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}

		// depth limit
		if depth(rel) > maxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// skip dirs
		if d.IsDir() {
			name := d.Name()
			switch name {
			case ".git", "vendor", "node_modules", "dist", "bin", "tmp":
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(out)
	return out, nil
}

func parseGoFile(path string) (FileInfo, bool) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		// если файл не парсится — просто пропускаем, чтобы не ломать дерево
		return FileInfo{}, false
	}

	fi := FileInfo{
		Path:  path,
		Pkg:   f.Name.Name,
		Types: map[string][]FuncInfo{},
	}

	// Соберём типы (чтобы красиво группировать методы)
	typeNames := map[string]bool{}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, sp := range gd.Specs {
			ts, ok := sp.(*ast.TypeSpec)
			if ok {
				typeNames[ts.Name.Name] = true
			}
		}
	}

	// Соберём функции/методы
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		sig := renderSignature(fd)

		recv := ""
		if fd.Recv != nil && len(fd.Recv.List) > 0 {
			recv = exprString(fd.Recv.List[0].Type)
			recv = strings.TrimPrefix(recv, "*")
		}

		info := FuncInfo{
			Name:      fd.Name.Name,
			Recv:      recv,
			Signature: sig,
		}

		if recv != "" && (typeNames[recv] || true) {
			fi.Types[recv] = append(fi.Types[recv], info)
		} else {
			fi.Funcs = append(fi.Funcs, info)
		}
	}

	// сортировки
	sort.Slice(fi.Funcs, func(i, j int) bool { return fi.Funcs[i].Name < fi.Funcs[j].Name })
	for t := range fi.Types {
		sort.Slice(fi.Types[t], func(i, j int) bool { return fi.Types[t][i].Name < fi.Types[t][j].Name })
	}

	return fi, true
}

func printTree(root string, infos []FileInfo) {
	// сгруппируем по директориям
	byDir := map[string][]FileInfo{}
	dirsSet := map[string]bool{}

	for _, fi := range infos {
		rel, _ := filepath.Rel(root, fi.Path)
		dir := filepath.Dir(rel)
		byDir[dir] = append(byDir[dir], fi)
		dirsSet[dir] = true
	}

	// соберём все директории в правильном порядке
	var dirs []string
	for d := range dirsSet {
		dirs = append(dirs, d)
	}
	sort.Slice(dirs, func(i, j int) bool {
		// корень первым
		if dirs[i] == "." {
			return true
		}
		if dirs[j] == "." {
			return false
		}
		return dirs[i] < dirs[j]
	})

	fmt.Printf("%s/\n", filepath.Base(root))

	for _, dir := range dirs {
		indent := strings.Repeat("│   ", max(0, depth(dir)-0))
		name := dir
		if name == "." {
			name = ""
		}

		if dir != "." {
			fmt.Printf("%s├── %s/\n", indent, filepath.Base(dir))
		}

		files := byDir[dir]
		sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

		for _, fi := range files {
			rel, _ := filepath.Rel(root, fi.Path)
			fileIndent := strings.Repeat("│   ", depth(filepath.Dir(rel)))
			fmt.Printf("%s│   ├── %s  (package %s)\n", fileIndent, filepath.Base(rel), fi.Pkg)

			// функции
			for _, fn := range fi.Funcs {
				fmt.Printf("%s│   │   ├── %s\n", fileIndent, compact(fn.Signature))
			}

			// методы по типам
			typeNames := make([]string, 0, len(fi.Types))
			for t := range fi.Types {
				typeNames = append(typeNames, t)
			}
			sort.Strings(typeNames)

			for _, t := range typeNames {
				fmt.Printf("%s│   │   ├── type %s методы:\n", fileIndent, t)
				for _, m := range fi.Types[t] {
					fmt.Printf("%s│   │   │   ├── %s\n", fileIndent, compact(m.Signature))
				}
			}
		}
	}
}

func renderSignature(fd *ast.FuncDecl) string {
	recv := ""
	if fd.Recv != nil && len(fd.Recv.List) > 0 {
		recv = "(" + exprString(fd.Recv.List[0].Type) + ") "
	}
	params := fieldListString(fd.Type.Params)
	results := fieldListString(fd.Type.Results)

	if results != "" {
		return fmt.Sprintf("func %s%s(%s) %s", recv, fd.Name.Name, params, results)
	}
	return fmt.Sprintf("func %s%s(%s)", recv, fd.Name.Name, params)
}

func fieldListString(fl *ast.FieldList) string {
	if fl == nil || len(fl.List) == 0 {
		return ""
	}
	var parts []string
	for _, f := range fl.List {
		t := exprString(f.Type)
		// имена параметров опустим, чтобы дерево было компактнее
		parts = append(parts, t)
	}
	// если несколько результатов, в Go они печатаются в скобках
	if fl == nil {
		return ""
	}
	if len(fl.List) > 1 && fl == fl { // просто чтобы не усложнять
		return "(" + strings.Join(parts, ", ") + ")"
	}
	return strings.Join(parts, ", ")
}

func exprString(e ast.Expr) string {
	if e == nil {
		return ""
	}
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.StarExpr:
		return "*" + exprString(x.X)
	case *ast.SelectorExpr:
		return exprString(x.X) + "." + x.Sel.Name
	case *ast.ArrayType:
		return "[]" + exprString(x.Elt)
	case *ast.MapType:
		return "map[" + exprString(x.Key) + "]" + exprString(x.Value)
	case *ast.Ellipsis:
		return "..." + exprString(x.Elt)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		return "func"
	default:
		return fmt.Sprintf("%T", e)
	}
}

func filterExportedFuncs(in []FuncInfo) []FuncInfo {
	out := make([]FuncInfo, 0, len(in))
	for _, f := range in {
		if ast.IsExported(f.Name) {
			out = append(out, f)
		}
	}
	return out
}

func compact(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func depth(rel string) int {
	if rel == "." || rel == "" {
		return 0
	}
	return len(strings.Split(rel, string(os.PathSeparator)))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
