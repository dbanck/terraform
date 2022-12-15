package lang

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/ext/tryfunc"
	ctyyaml "github.com/zclconf/go-cty-yaml"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	"github.com/hashicorp/terraform/internal/experiments"
	"github.com/hashicorp/terraform/internal/lang/funcs"
)

var impureFunctions = []string{
	"bcrypt",
	"timestamp",
	"uuid",
}

// Functions returns the set of functions that should be used to when evaluating
// expressions in the receiving scope.
func (s *Scope) Functions() map[string]function.Function {
	s.funcsLock.Lock()
	if s.funcs == nil {
		// Some of our functions are just directly the cty stdlib functions.
		// Others are implemented in the subdirectory "funcs" here in this
		// repository. New functions should generally start out their lives
		// in the "funcs" directory and potentially graduate to cty stdlib
		// later if the functionality seems to be something domain-agnostic
		// that would be useful to all applications using cty functions.

		s.funcs = map[string]function.Function{
			"abs":              funcs.WithDescription("abs", stdlib.AbsoluteFunc),
			"abspath":          funcs.AbsPathFunc,
			"alltrue":          funcs.AllTrueFunc,
			"anytrue":          funcs.AnyTrueFunc,
			"basename":         funcs.BasenameFunc,
			"base64decode":     funcs.Base64DecodeFunc,
			"base64encode":     funcs.Base64EncodeFunc,
			"base64gzip":       funcs.Base64GzipFunc,
			"base64sha256":     funcs.Base64Sha256Func, // TODO!
			"base64sha512":     funcs.Base64Sha512Func, // TODO!
			"bcrypt":           funcs.BcryptFunc,
			"can":              tryfunc.CanFunc, // TODO!
			"ceil":             funcs.WithDescription("ceil", stdlib.CeilFunc),
			"chomp":            funcs.WithDescription("chomp", stdlib.ChompFunc),
			"cidrhost":         funcs.CidrHostFunc,
			"cidrnetmask":      funcs.CidrNetmaskFunc,
			"cidrsubnet":       funcs.CidrSubnetFunc,
			"cidrsubnets":      funcs.CidrSubnetsFunc,
			"coalesce":         funcs.CoalesceFunc,
			"coalescelist":     funcs.WithDescription("coalescelist", stdlib.CoalesceListFunc),
			"compact":          funcs.WithDescription("compact", stdlib.CompactFunc),
			"concat":           funcs.WithDescription("concat", stdlib.ConcatFunc),
			"contains":         funcs.WithDescription("contains", stdlib.ContainsFunc),
			"csvdecode":        funcs.WithDescription("csvdecode", stdlib.CSVDecodeFunc),
			"dirname":          funcs.DirnameFunc,
			"distinct":         funcs.WithDescription("distinct", stdlib.DistinctFunc),
			"element":          funcs.WithDescription("element", stdlib.ElementFunc),
			"endswith":         funcs.EndsWithFunc,
			"chunklist":        funcs.WithDescription("chunklist", stdlib.ChunklistFunc),
			"file":             funcs.MakeFileFunc(s.BaseDir, false),      // TODO!
			"fileexists":       funcs.MakeFileExistsFunc(s.BaseDir),       // TODO!
			"fileset":          funcs.MakeFileSetFunc(s.BaseDir),          // TODO!
			"filebase64":       funcs.MakeFileFunc(s.BaseDir, true),       // TODO!
			"filebase64sha256": funcs.MakeFileBase64Sha256Func(s.BaseDir), // TODO!
			"filebase64sha512": funcs.MakeFileBase64Sha512Func(s.BaseDir), // TODO!
			"filemd5":          funcs.MakeFileMd5Func(s.BaseDir),          // TODO!
			"filesha1":         funcs.MakeFileSha1Func(s.BaseDir),         // TODO!
			"filesha256":       funcs.MakeFileSha256Func(s.BaseDir),       // TODO!
			"filesha512":       funcs.MakeFileSha512Func(s.BaseDir),       // TODO!
			"flatten":          funcs.WithDescription("flatten", stdlib.FlattenFunc),
			"floor":            funcs.WithDescription("floor", stdlib.FloorFunc),
			"format":           funcs.WithDescription("format", stdlib.FormatFunc),
			"formatdate":       funcs.WithDescription("formatdate", stdlib.FormatDateFunc),
			"formatlist":       funcs.WithDescription("formatlist", stdlib.FormatListFunc),
			"indent":           funcs.WithDescription("indent", stdlib.IndentFunc),
			"index":            funcs.IndexFunc, // stdlib.IndexFunc is not compatible
			"join":             funcs.WithDescription("join", stdlib.JoinFunc),
			"jsondecode":       funcs.WithDescription("jsondecode", stdlib.JSONDecodeFunc),
			"jsonencode":       funcs.WithDescription("jsonencode", stdlib.JSONEncodeFunc),
			"keys":             funcs.WithDescription("keys", stdlib.KeysFunc),
			"length":           funcs.LengthFunc,
			"list":             funcs.ListFunc,
			"log":              funcs.WithDescription("log", stdlib.LogFunc),
			"lookup":           funcs.LookupFunc,
			"lower":            funcs.WithDescription("lower", stdlib.LowerFunc),
			"map":              funcs.MapFunc,
			"matchkeys":        funcs.MatchkeysFunc,
			"max":              funcs.WithDescription("max", stdlib.MaxFunc),
			"md5":              funcs.Md5Func, // TODO!
			"merge":            funcs.WithDescription("merge", stdlib.MergeFunc),
			"min":              funcs.WithDescription("min", stdlib.MinFunc),
			"one":              funcs.OneFunc,
			"parseint":         funcs.WithDescription("parseint", stdlib.ParseIntFunc),
			"pathexpand":       funcs.PathExpandFunc,
			"pow":              funcs.WithDescription("pow", stdlib.PowFunc),
			"range":            funcs.WithDescription("range", stdlib.RangeFunc),
			"regex":            funcs.WithDescription("regex", stdlib.RegexFunc),
			"regexall":         funcs.WithDescription("regexall", stdlib.RegexAllFunc),
			"replace":          funcs.ReplaceFunc,
			"reverse":          funcs.WithDescription("reverse", stdlib.ReverseListFunc),
			"rsadecrypt":       funcs.RsaDecryptFunc,
			"sensitive":        funcs.SensitiveFunc,
			"nonsensitive":     funcs.NonsensitiveFunc,
			"setintersection":  funcs.WithDescription("setintersection", stdlib.SetIntersectionFunc),
			"setproduct":       funcs.WithDescription("setproduct", stdlib.SetProductFunc),
			"setsubtract":      funcs.WithDescription("setsubtract", stdlib.SetSubtractFunc),
			"setunion":         funcs.WithDescription("setunion", stdlib.SetUnionFunc),
			"sha1":             funcs.Sha1Func,   // TODO!
			"sha256":           funcs.Sha256Func, // TODO!
			"sha512":           funcs.Sha512Func, // TODO!
			"signum":           funcs.WithDescription("signum", stdlib.SignumFunc),
			"slice":            funcs.WithDescription("slice", stdlib.SliceFunc),
			"sort":             funcs.WithDescription("sort", stdlib.SortFunc),
			"split":            funcs.WithDescription("split", stdlib.SplitFunc),
			"startswith":       funcs.StartsWithFunc,
			"strrev":           funcs.WithDescription("strrev", stdlib.ReverseFunc),
			"substr":           funcs.WithDescription("substr", stdlib.SubstrFunc),
			"sum":              funcs.SumFunc,
			"textdecodebase64": funcs.TextDecodeBase64Func,
			"textencodebase64": funcs.TextEncodeBase64Func,
			"timestamp":        funcs.TimestampFunc,
			"timeadd":          funcs.WithDescription("timeadd", stdlib.TimeAddFunc),
			"timecmp":          funcs.TimeCmpFunc,
			"title":            funcs.WithDescription("title", stdlib.TitleFunc),
			"tostring":         funcs.MakeToFunc(cty.String),                      // TODO!
			"tonumber":         funcs.MakeToFunc(cty.Number),                      // TODO!
			"tobool":           funcs.MakeToFunc(cty.Bool),                        // TODO!
			"toset":            funcs.MakeToFunc(cty.Set(cty.DynamicPseudoType)),  // TODO!
			"tolist":           funcs.MakeToFunc(cty.List(cty.DynamicPseudoType)), // TODO!
			"tomap":            funcs.MakeToFunc(cty.Map(cty.DynamicPseudoType)),  // TODO!
			"transpose":        funcs.TransposeFunc,
			"trim":             funcs.WithDescription("trim", stdlib.TrimFunc),
			"trimprefix":       funcs.WithDescription("trimprefix", stdlib.TrimPrefixFunc),
			"trimspace":        funcs.WithDescription("trimspace", stdlib.TrimSpaceFunc),
			"trimsuffix":       funcs.WithDescription("trimsuffix", stdlib.TrimSuffixFunc),
			"try":              tryfunc.TryFunc, // TODO!
			"upper":            funcs.WithDescription("upper", stdlib.UpperFunc),
			"urlencode":        funcs.URLEncodeFunc,
			"uuid":             funcs.UUIDFunc,
			"uuidv5":           funcs.UUIDV5Func,
			"values":           funcs.WithDescription("values", stdlib.ValuesFunc),
			"yamldecode":       funcs.WithDescription("yamldecode", ctyyaml.YAMLDecodeFunc),
			"yamlencode":       funcs.WithDescription("yamlencode", ctyyaml.YAMLEncodeFunc),
			"zipmap":           funcs.WithDescription("zipmap", stdlib.ZipmapFunc),
		}

		s.funcs["templatefile"] = funcs.MakeTemplateFileFunc(s.BaseDir, func() map[string]function.Function {
			// The templatefile function prevents recursive calls to itself
			// by copying this map and overwriting the "templatefile" entry.
			return s.funcs
		})

		if s.ConsoleMode {
			// The type function is only available in terraform console.
			s.funcs["type"] = funcs.TypeFunc
		}

		if s.PureOnly {
			// Force our few impure functions to return unknown so that we
			// can defer evaluating them until a later pass.
			for _, name := range impureFunctions {
				s.funcs[name] = function.Unpredictable(s.funcs[name])
			}
		}
	}
	s.funcsLock.Unlock()

	return s.funcs
}

// experimentalFunction checks whether the given experiment is enabled for
// the recieving scope. If so, it will return the given function verbatim.
// If not, it will return a placeholder function that just returns an
// error explaining that the function requires the experiment to be enabled.
//
//lint:ignore U1000 Ignore unused function error for now
func (s *Scope) experimentalFunction(experiment experiments.Experiment, fn function.Function) function.Function {
	if s.activeExperiments.Has(experiment) {
		return fn
	}

	err := fmt.Errorf(
		"this function is experimental and available only when the experiment keyword %s is enabled for the current module",
		experiment.Keyword(),
	)

	return function.New(&function.Spec{
		Params:   fn.Params(),
		VarParam: fn.VarParam(),
		Type: func(args []cty.Value) (cty.Type, error) {
			return cty.DynamicPseudoType, err
		},
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			// It would be weird to get here because the Type function always
			// fails, but we'll return an error here too anyway just to be
			// robust.
			return cty.DynamicVal, err
		},
	})
}
