package interp

import (
	"path"
)

// Gta performs a global types analysis on the AST, registering types,
// variables and functions at package level, prior to CFG. All function
// bodies are skipped.
// GTA is necessary to handle out of order declarations and multiple
// source files packages.
func (interp *Interpreter) Gta(root *Node) {
	var pkgName string
	scope := interp.universe

	root.Walk(func(n *Node) bool {
		switch n.kind {
		case Define:
			varName := n.child[0].ident
			scope.sym[varName] = &Symbol{kind: Var, global: true, index: scope.inc(interp)}
			if len(n.child) > 1 {
				scope.sym[varName].typ = nodeType(interp, scope, n.child[1])
			} else {
				scope.sym[varName].typ = nodeType(interp, scope, n.anc.child[0].child[1])
			}
			return false

		case File:
			pkgName = n.child[0].ident
			if _, ok := interp.scope[pkgName]; !ok {
				interp.scope[pkgName] = scope.push(0)
			}
			scope = interp.scope[pkgName]

		case FuncDecl:
			n.typ = nodeType(interp, scope, n.child[2])
			scope.sym[n.child[1].ident] = &Symbol{kind: Func, typ: n.typ, node: n}
			if len(n.child[0].child) > 0 {
				// function is a method, add it to the related type
				var t *Type
				var typeName string
				n.ident = n.child[1].ident
				recv := n.child[0].child[0]
				if len(recv.child) < 2 {
					// Receiver var name is skipped in method declaration (fix that in AST ?)
					typeName = recv.child[0].ident
				} else {
					typeName = recv.child[1].ident
				}
				if typeName == "" {
					typeName = recv.child[1].child[0].ident
					elemtype := scope.getType(typeName)
					t = &Type{cat: PtrT, val: elemtype}
					elemtype.method = append(elemtype.method, n)
				} else {
					t = scope.getType(typeName)
				}
				t.method = append(t.method, n)
			}
			return false

		case ImportSpec:
			var name, ipath string
			if len(n.child) == 2 {
				ipath = n.child[1].val.(string)
				name = n.child[0].ident
			} else {
				ipath = n.child[0].val.(string)
				name = path.Base(ipath)
			}
			if pkg, ok := interp.binValue[ipath]; ok {
				if name == "." {
					for n, s := range pkg {
						scope.sym[n] = &Symbol{typ: &Type{cat: BinT}, val: s}
					}
				} else {
					scope.sym[name] = &Symbol{typ: &Type{cat: BinPkgT}, path: ipath}
				}
			} else {
				// TODO: make sure we do not import a src package more than once
				interp.importSrcFile(ipath)
				scope.sym[name] = &Symbol{typ: &Type{cat: SrcPkgT}, path: ipath}
			}

		case TypeSpec:
			typeName := n.child[0].ident
			if n.child[1].kind == Ident {
				n.typ = &Type{cat: AliasT, val: nodeType(interp, scope, n.child[1])}
			} else {
				n.typ = nodeType(interp, scope, n.child[1])
			}
			// Type may already be declared for a receiver in a method function
			if scope.sym[typeName] == nil {
				scope.sym[typeName] = &Symbol{kind: Typ}
			}
			scope.sym[typeName].typ = n.typ
			return false

		}
		return true
	}, nil)
}