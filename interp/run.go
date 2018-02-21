package interp

import (
	"fmt"
)

// Function to run at CFG execution
type Builtin func(n *Node, f *Frame)

var builtin = [...]Builtin{
	Nop:      nop,
	ArrayLit: arrayLit,
	Assign:   assign,
	AssignX:  assignX,
	Add:      add,
	And:      and,
	Call:     call,
	Case:     _case,
	Dec:      nop,
	Equal:    equal,
	Greater:  greater,
	GetIndex: getIndex,
	Inc:      inc,
	Land:     land,
	Lor:      lor,
	Lower:    lower,
	Range:    _range,
	Return:   _return,
	Sub:      sub,
}

// Run a Go function
func Run(def *Node, cf *Frame, args []*Node, rets []int) {
	//fmt.Println("run", def.Child[0].ident)
	// Allocate a new Frame to store local variables
	f := Frame(make([]interface{}, def.findex))

	// Pass func parameters by value: copy each parameter from caller frame
	param := def.Child[1].Child[0].Child
	for i, arg := range args {
		f[param[i].findex] = value(arg, cf)
	}
	//fmt.Println("frame:", f)

	// Execute by walking the CFG and running node func at each step
	body := def.Child[2]
	for n := body.Start; n != nil; {
		n.run(n, &f)
		//fmt.Println("run", n.index, n.kind, n.action, value(n, &f))
		if n.fnext == nil || value(n, &f).(bool) {
			n = n.tnext
		} else {
			n = n.fnext
		}
	}

	// Propagate return values to caller frame
	for i, ret := range rets {
		(*cf)[ret] = f[i]
	}
}

// Functions set to run during execution of CFG

func value(n *Node, f *Frame) interface{} {
	switch n.kind {
	case BasicLit, FuncDecl:
		return n.val
	default:
		return (*f)[n.findex]
	}
}

// AssignX(n, f) implements assignement for a single call which returns multiple values
func assignX(n *Node, f *Frame) {
	l := len(n.Child) - 1
	b := n.Child[l].findex
	for i, c := range n.Child[:l] {
		(*f)[c.findex] = (*f)[b+i]
	}
}

// Assign implements assignement with the same number of left and right values
func assign(n *Node, f *Frame) {
	l := len(n.Child) / 2
	for i, c := range n.Child[:l] {
		(*f)[c.findex] = value(n.Child[l+i], f)
	}
}

func and(n *Node, f *Frame) {
	(*f)[n.findex] = value(n.Child[0], f).(int) & value(n.Child[1], f).(int)
}

func printa(n []*Node, f *Frame) {
	for i, m := range n {
		if i > 0 {
			fmt.Printf(" ")
		}
		fmt.Printf("%v", value(m, f))
	}
	fmt.Println("")
}

//func (interp *Interpreter) call(n *Node, f *Frame) {
func call(n *Node, f *Frame) {
	//fmt.Println("call", n.Child[0].ident)
	// FIXME: builtin detection should be done at CFG generation
	if n.Child[0].ident == "println" {
		printa(n.Child[1:], f)
		return
	}
	fn := n.val.(*Node)
	var rets []int
	if len(fn.Child[1].Child) > 1 {
		if fieldList := fn.Child[1].Child[1]; fieldList != nil {
			rets = make([]int, len(fieldList.Child))
			for i, _ := range fieldList.Child {
				rets[i] = n.findex + i
			}
		}
	}
	Run(fn, f, n.Child[1:], rets)
}

func getIndex(n *Node, f *Frame) {
	a := value(n.Child[0], f).([]interface{})
	(*f)[n.findex] = a[value(n.Child[1], f).(int)]
}

func add(n *Node, f *Frame) {
	(*f)[n.findex] = value(n.Child[0], f).(int) + value(n.Child[1], f).(int)
}

func sub(n *Node, f *Frame) {
	(*f)[n.findex] = value(n.Child[0], f).(int) - value(n.Child[1], f).(int)
}

func equal(n *Node, f *Frame) {
	(*f)[n.findex] = value(n.Child[0], f) == value(n.Child[1], f)
}

func inc(n *Node, f *Frame) {
	(*f)[n.findex] = value(n.Child[0], f).(int) + 1
}

func greater(n *Node, f *Frame) {
	(*f)[n.findex] = value(n.Child[0], f).(int) > value(n.Child[1], f).(int)
}

func land(n *Node, f *Frame) {
	if v := value(n.Child[0], f).(bool); v {
		(*f)[n.findex] = value(n.Child[1], f).(bool)
	} else {
		(*f)[n.findex] = v
	}
}

func lor(n *Node, f *Frame) {
	if v := value(n.Child[0], f).(bool); v {
		(*f)[n.findex] = v
	} else {
		(*f)[n.findex] = value(n.Child[1], f).(bool)
	}
}

func lower(n *Node, f *Frame) {
	(*f)[n.findex] = value(n.Child[0], f).(int) < value(n.Child[1], f).(int)
}

func nop(n *Node, f *Frame) {}

func _return(n *Node, f *Frame) {
	for i, c := range n.Child {
		(*f)[i] = value(c, f)
	}
	// FIXME: should be done during compiling, not run
	n.tnext = nil
}

// create an array of litteral values
func arrayLit(n *Node, f *Frame) {
	a := make([]interface{}, len(n.Child)-1)
	for i, c := range n.Child[1:] {
		a[i] = value(c, f)
	}
	(*f)[n.findex] = a
}

func _range(n *Node, f *Frame) {
	i, index := 0, n.Child[0].findex
	if (*f)[index] != nil {
		i = (*f)[index].(int)
	}
	a := value(n.Child[2], f).([]interface{})
	if i >= len(a) {
		(*f)[n.findex] = false
		return
	}
	(*f)[index] = i + 1
	(*f)[n.Child[1].findex] = a[i]
	(*f)[n.findex] = true
}

func _case(n *Node, f *Frame) {
	if len(n.Child) == 1 {
		//(*f)[n.findex] = true
		return
	}
	(*f)[n.findex] = value(n.anc.anc.Child[0], f) == value(n.Child[0], f)
}