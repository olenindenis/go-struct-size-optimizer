package analyzer

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// WriteMode instructs the analyzer to apply edits directly to source files
// instead of only reporting diagnostics. Set by the CLI via -w flag.
var WriteMode bool

var Analyzer = &analysis.Analyzer{
	Name: "structalign",
	Doc:  "optimize struct field alignment",
	Run:  run,
}

type fieldInfo struct {
	name  string
	typ   types.Type
	size  int64
	align int64
	field *ast.Field
}

func run(pass *analysis.Pass) (any, error) {
	if pass.TypesInfo == nil {
		return nil, nil
	}
	sizes := pass.TypesSizes

	for _, file := range pass.Files {
		ignoreLines := buildIgnoreLines(pass.Fset, file)

		ast.Inspect(file, func(n ast.Node) bool {
			st, ok := n.(*ast.StructType)
			if !ok {
				return true
			}

			if isIgnored(pass.Fset, ignoreLines, st) {
				return true
			}

			var fields []fieldInfo

			for _, f := range st.Fields.List {
				t := pass.TypesInfo.TypeOf(f.Type)
				if t == nil {
					continue
				}

				if hasTags(f) {
					return true
				}

				elemSize := sizes.Sizeof(t)
				align := sizes.Alignof(t)

				var name string
				var totalSize int64
				if len(f.Names) == 0 {
					name = types.TypeString(t, nil)
					totalSize = elemSize
				} else {
					name = f.Names[0].Name
					totalSize = elemSize * int64(len(f.Names))
				}

				fields = append(fields, fieldInfo{
					name:  name,
					typ:   t,
					size:  totalSize,
					align: align,
					field: f,
				})
			}

			if len(fields) == 0 {
				return true
			}

			origSize := calcStructSize(fields)
			optimized := optimizeFields(fields)
			optSize := calcStructSize(optimized)

			if optSize < origSize {
				origText, err1 := renderNode(pass, st)
				fixedText, err2 := buildFixedStruct(optimized)
				if err1 != nil || err2 != nil {
					return true
				}

				difference := buildDiff(string(origText), string(fixedText))

				pass.Report(analysis.Diagnostic{
					Pos: st.Pos(),
					End: st.End(),
					Message: fmt.Sprintf(
						"struct can be optimized: %d -> %d bytes\nDiff:\n%s",
						origSize,
						optSize,
						difference,
					),
					SuggestedFixes: []analysis.SuggestedFix{
						{
							Message: "Reorder struct fields",
							TextEdits: []analysis.TextEdit{
								{
									Pos:     st.Pos(),
									End:     st.End(),
									NewText: fixedText,
								},
							},
						},
					},
				})

				if WriteMode {
					addEdit(pass, st.Pos(), st.End(), fixedText)
				}
			}

			return true
		})
	}

	return nil, nil
}

func calcStructSize(fields []fieldInfo) int64 {
	var offset, maxAlign int64
	maxAlign = 1

	for _, f := range fields {
		if f.align > maxAlign {
			maxAlign = f.align
		}
		padding := (f.align - (offset % f.align)) % f.align
		offset += padding + f.size
	}

	padding := (maxAlign - (offset % maxAlign)) % maxAlign
	return offset + padding
}
