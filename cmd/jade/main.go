package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/Joker/jade"
	"golang.org/x/tools/imports"
)

var (
	dict     = map[string]string{}
	lib_name = ""
	outdir   string
	basedir  string
	pkg_name string
	stdlib   bool
	stdbuf   bool
	writer   bool
	inline   bool
	format   bool
	verbose  bool
	ns_files = map[string]bool{}
)

func use() {
	fmt.Printf("Usage: %s [OPTION]... [FILE]... \n", os.Args[0])
	flag.PrintDefaults()
}
func init() {
	flag.StringVar(&outdir, "d", "", `directory for generated .go files`)
	flag.StringVar(&basedir, "basedir", "./", `base directory for templates`)
	flag.StringVar(&pkg_name, "pkg", "", `package name for generated files`)
	flag.BoolVar(&format, "fmt", true, `HTML pretty print output for generated functions`)
	flag.BoolVar(&inline, "inline", true, `inline HTML in generated functions`)
	flag.BoolVar(&stdlib, "stdlib", false, `use stdlib functions`)
	flag.BoolVar(&stdbuf, "stdbuf", true, `use bytes.Buffer  [default bytebufferpool.ByteBuffer]`)
	flag.BoolVar(&writer, "writer", true, `use io.Writer for output`)
	flag.BoolVar(&verbose, "v", false, `increase log output`)

	log.SetFlags(log.Lmsgprefix)
}

//

type goAST struct {
	node *ast.File
	fset *token.FileSet
}

func (a *goAST) bytes(bb *bytes.Buffer) []byte {
	printer.Fprint(bb, a.fset, a.node)
	return bb.Bytes()
}

func parseGoSrc(fileName string, GoSrc interface{}) (out goAST, err error) {
	out.fset = token.NewFileSet()
	out.node, err = parser.ParseFile(out.fset, fileName, GoSrc, parser.ParseComments)
	return
}

func goImports(absPath string, src []byte) []byte {
	fmtOut, err := imports.Process(absPath, src, &imports.Options{TabWidth: 4, TabIndent: true, Comments: true, Fragment: true})
	if err != nil {
		log.Fatalln("goImports(): ", err)
	}
	return fmtOut
}

//

func genFile(path, outdir, pkg_name string) {
	if verbose {
		log.Printf("\nInput file: %s\n", path)
	}

	var (
		dir, fname = filepath.Split(path)
		outPath    = outdir + "/" + fname
		rx, _      = regexp.Compile("[^a-zA-Z0-9]+")
		constName  = rx.ReplaceAllString(fname[:len(fname)-4], "")
	)

	wd, err := os.Getwd()
	if err == nil && wd != dir && dir != "" {
		os.Chdir(dir)
		defer os.Chdir(wd)
	}

	if _, ok := ns_files[fname]; ok {
		sfx := "_" + strconv.Itoa(len(ns_files))
		ns_files[fname+sfx] = true
		outPath += sfx
		constName += sfx
	} else {
		ns_files[fname] = true
	}

	fl, err := os.ReadFile(fname)
	if err != nil {
		log.Fatalln("cmd/jade: ReadFile(): ", err)
	}

	//

	jst, err := jade.New(path).Parse(fl)
	if err != nil {
		log.Fatalln("cmd/jade: jade.New(path).Parse(): ", err)
	}

	var (
		bb  = new(bytes.Buffer)
		tpl = newLayout(constName)
	)
	tpl.writeBefore(bb)
	before := bb.Len()
	jst.WriteIn(bb)
	if before == bb.Len() {
		if verbose {
			fmt.Print("generated: skipped (empty output)  done.\n\n")
		}
		return
	}
	tpl.writeAfter(bb)

	//

	gst, err := parseGoSrc(outPath, bb)
	if err != nil {
		// write error to stderr for redirection where you want
		fmt.Fprint(os.Stderr, string(bb.Bytes()))
		//ioutil.WriteFile(outPath+"__Error.go", bb.Bytes(), 0644)
		fmt.Fprintf(os.Stdout, "Error: Partially completed template available on stderr.\ncmd/jade: parseGoSrc(): %s\n", err)
		os.Exit(1)
	}

	gst.collapseWriteString(inline, constName)
	gst.checkType()
	gst.checkUnresolvedBlock()

	bb.Reset()
	fmtOut := goImports(outPath, gst.bytes(bb))

	//

	outfile := filepath.Clean(outPath + ".go")
	err = os.WriteFile(outfile, fmtOut, 0644)
	if err != nil {
		log.Fatalln("cmd/jade: WriteFile(): ", err)
	}
	if verbose {
		log.Printf("Generated %s\n", outfile)
	}
}

func genDir(dir, outdir, pkg_name string) {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("prevent panic by handling failure accessing a path %q: %v\n", dir, err)
		}

		if ext := filepath.Ext(info.Name()); ext == ".jade" || ext == ".pug" {
			genFile(path, outdir, pkg_name)
		}
		return nil
	})
	if err != nil {
		log.Fatalln(err)
	}
}

//

func main() {
	flag.Usage = use
	flag.Parse()
	if len(flag.Args()) == 0 {
		use()
		return
	}

	jade.Config(golang)

	if outdir != "" {
		if _, err := os.Stat(outdir); os.IsNotExist(err) {
			os.MkdirAll(outdir, 0755)
		}
		outdir, _ = filepath.Abs(outdir)
	}

	if _, err := os.Stat(basedir); !os.IsNotExist(err) && basedir != "./" {
		os.Chdir(basedir)
	}

	for _, jadePath := range flag.Args() {

		stat, err := os.Stat(jadePath)
		if err != nil {
			log.Fatalln(err)
		}

		absPath, _ := filepath.Abs(jadePath)

		// default to generate next to jade file
		if outdir == "" {
			outdir, _ = filepath.Split(absPath)
		}
		// guess package name from directory structure
		if pkg_name == "" {
			pkg_name = filepath.Base(outdir)
		}
		// defaults if all else fails
		if pkg_name == "." {
			pkg_name = "jade"
		}
		if len(outdir) == 0 {
			outdir = "./"
		}
		if verbose {
			fmt.Printf("Package: %s\nOutput directory: %s\n", pkg_name, outdir)
		}

		if stat.IsDir() {
			genDir(absPath, outdir, pkg_name)
		} else {
			genFile(absPath, outdir, pkg_name)
		}
		if !stdlib {
			makeJfile(stdbuf)
		}
	}
}
