package tree

import (
	"bytes"
	"fmt"
	"net/http"
)

const (
	staticNode = iota
	paramNode
)

type Param struct {
	Key   string
	Value string
}

type Node struct {
	parent   *Node
	children []*Node
	prefix   string
	value    int64
	nodeType int
	param    Param
	handler  http.HandlerFunc
}

func NewNode() *Node {
	return &Node{
		parent:   nil,
		children: make([]*Node, 0),
	}
}

func (n *Node) Insert(str string, handler http.HandlerFunc) {
	var _n *Node

	suffix := str
	commonlength := lcp(n.prefix, suffix)

	// rootが子ノードを持っていない時
	if len(n.children) == 0 {
		newNode := &Node{
			parent:   n,
			prefix:   "/",
			handler:  nil,
			nodeType: staticNode,
		}
		n.children = append(n.children, newNode)
		suffix = suffix[commonlength:]
		_n = newNode
		// fmt.Printf("insert %v, after %v\n", newNode, n)
	} else {
		_n = n.children[0]
	}

	// pathが'/'の時
	if suffix == "/" {
		_n.handler = handler
		return
	}

	for {
		if _n != nil {
			n = _n
		}

		commonlength := lcp(n.prefix, suffix)

		// 完全一致しない場合
		if len(suffix) > commonlength {
			mn := len(suffix)
			var _next *Node

			// children の中に prefix と部分一致するものがあるか探す
			for i := 0; i < len(n.children); i++ {
				l := lcp(n.children[i].prefix, suffix)
				if l <= mn && l != 0 {
					mn = l
					_next = n.children[i]
				}
			}

			// 部分一致するものがあって、/の場合は、次のノードにする
			if mn < len(suffix) && mn > 0 && suffix[0] == '/' {
				// 中間ノードがある場合、次のノードにする
				if _next.prefix == suffix[:mn] {
					suffix = suffix[mn:]
					_n = _next
					continue
				}
			}

			// 部分一致するものがある場合で、パス内の文字列の場合
			if mn < len(suffix) && mn > 0 && suffix[0] != '/' {
				// 中間ノードがある場合、次のノードにする
				if _next.prefix == suffix[:mn] {
					suffix = suffix[mn:]
					_n = _next
					continue
				}

				// 中間ノードが必要な場合は作成して次のノードにする
				interemediate := &Node{
					parent:   n,
					prefix:   suffix[:mn],
					children: make([]*Node, 0),
					nodeType: staticNode,
				}

				interemediate.parent = n
				for i := 0; i < len(n.children); i++ {
					if suffix[:mn] == n.children[i].prefix[:mn] {
						// 中間ノードに子ノードを追加する
						interemediate.children = append(interemediate.children, n.children[i])

						// 子ノードの親ノードを中間ノードに更新する
						n.children[i].parent = interemediate
						n.children[i].prefix = n.children[i].prefix[mn:]

						// 親ノードに存在する既存のchildrenを削除する
						n.children = append(n.children[:i], n.children[i+1:]...)
					}
				}

				// 親ノードに中間ノードを追加する
				n.children = append(n.children, interemediate)
				suffix = suffix[mn:]
				_n = interemediate
				continue
			}
		}

		if len(suffix) == 0 {
			return
		}

		// 既に子ノードにパラメータノードがないか探す
		if suffix[0] == ':' {
			i := 0

			// パラメータだけを切り出す
			for ; i < len(suffix); i++ {
				if suffix[i] == '/' {
					break
				}
			}

			var _child *Node

			// 既にパラメータノードがあるか探す
			for i := 0; i < len(n.children); i++ {
				if n.children[i].nodeType == paramNode {
					_child = n.children[i]
					break
				}
			}

			if _child != nil {
				l := lcp(_child.prefix, suffix[:i])

				// 既にあるパラメータノードと一致しない場合は、panicを返す
				if l != len(_child.prefix) {
					panic("param node is already exist")
				}

				// suffixが一致する場合は、ハンドラを設定して終了
				if l == len(suffix) {
					_child.handler = handler
					return
				}

				// suffixが一致しない場合は次のノードとする
				_n = _child
				suffix = suffix[i:]
				continue
			}
		}

		// パスパラメータの部分だけ切り出してノードを作成する
		if commonlength == 0 && suffix[0] == ':' {
			// パラメータ部分だけ切り出す
			var paramBytes bytes.Buffer
			i := 1
			for ; i < len(suffix); i++ {
				if suffix[i] == '/' {
					break
				}
				paramBytes.WriteByte(suffix[i])
			}

			// 新規にパラメータのノードを作成する
			newParamNode := &Node{
				parent:   n,
				prefix:   suffix[:i],
				param:    Param{Key: paramBytes.String(), Value: ""},
				children: make([]*Node, 0),
				nodeType: paramNode,
			}

			// 最後のノードの場合はハンドラを設定する
			if i == len(suffix) {
				newParamNode.handler = handler
			}

			n.children = append(n.children, newParamNode)
			fmt.Printf("insert2 %v, after %v\n", newParamNode, n)
			suffix = suffix[i:]
			_n = newParamNode
			continue
		}

		// それ以上の最大共通接頭辞がない場合は、新規ノードを作成する
		if commonlength == 0 {
			i := 0
			for ; i < len(suffix); i++ {
				if suffix[i] == '/' {
					break
				}
			}
			newNode := &Node{
				parent:   n,
				prefix:   suffix[:i],
				handler:  handler,
				nodeType: staticNode,
			}
			suffix = suffix[i:]
			n.children = append(n.children, newNode)
			// fmt.Printf("insert %v, after %v\n", newNode, n)
			_n = newNode
			continue
		}

		// "/"の場合は新規にノードを作成する
		if suffix[:commonlength] == "/" && n.prefix != "/" {
			newNode := &Node{
				parent:   n,
				prefix:   suffix[:commonlength],
				nodeType: staticNode,
			}
			n.children = append(n.children, newNode)
			suffix = suffix[commonlength:]
			_n = newNode
			continue
		}

		// 共通接頭辞と次の検索するノードの更新
		suffix = suffix[commonlength:]
		_n = n
	}
}

func (n *Node) Search(path string) (http.HandlerFunc, []*Param) {
	_n := n
	// 前のノードを保存する
	var _prev *Node
	var params []*Param
	now := ""
	suffix := ""

	for {
		if _prev != nil {
			_n = _prev
		}
		if len(path) > len(now) {
			suffix = path[len(now):]
		}

		_n, tnow := staticSearch(_n, suffix)
		now += tnow

		// 完全一致するパスがあるため、ハンドラを返す
		if now == path {
			return _n.handler, params
		}

		// ここまでくる場合は、完全一致していないため backtrack でパラメータノードを子供に持ったノードまで遡る
		_n, now = backtrack(_n, now)

		// パスパラメータルーティングを行う
		_n, tnow, tparams := paramSearch(_n, path[len(now):])

		// パスの更新
		now += tnow

		params = append(params, tparams...)

		if now == path {
			return _n.handler, params
		}

		// ノードが更新されていない = もう検索ができない ので nil を返す
		if _n == _prev {
			return nil, params
		}
		_prev = _n
	}
}

func lcp(a, b string) int {
	for i := 0; i < min(len(a), len(b)); i++ {
		if b[i] == '/' {
			return min(i+1, len(b))
		}
		if a[i] != b[i] {
			return i
		}
	}
	return min(len(a), len(b))
}

func staticSearch(n *Node, path string) (*Node, string) {
	suffix := path
	var _n *Node

	// pathが"/"の時
	if path == "/" {
		return n.children[0], "/"
	}

	// 現在のパスを保持し、入力されたパスと同じかどうかを確認する
	now := ""

	for {
		// 次のノードが設定されている時
		if _n != nil {
			n = _n
		}

		// 処理する文字列が無い時
		if suffix == "" && now == path {
			return n, now
		}

		// 今のノードの子ノードから最大共通接頭辞が長いものを次のノードにする
		mx := 0
		for i := 0; i < len(n.children); i++ {
			l := lcp(n.children[i].prefix, suffix)
			// ここにstaticNodeの条件を追加
			if l > mx && n.children[i].nodeType == staticNode {
				_n = n.children[i]
				mx = l
			}
		}

		// 接頭辞の長さが0の場合は、次の値がないので今のノードとマッチした文字列の値を返す
		if mx == 0 {
			return n, now
		}

		// 処理するsuffixを更新する
		suffix = suffix[mx:]
		// 現在のpathを更新する
		now += _n.prefix

		// 次のノードが更新されているため、処理を続ける
		if mx > 0 {
			continue
		}
	}
}

func backtrack(n *Node, path string) (*Node, string) {
	var _n *Node
	for {
		if _n != nil {
			n = _n
		}
		// 子ノードにパラメータノードがある場合は、パラメータノードとパスを返す
		for i := 0; i < len(n.children); i++ {
			if n.children[i].nodeType == paramNode {
				return n.children[i], path
			}
		}

		_n = n.parent

		// ノードに応じて path の値を巻き戻す
		if len(path) > len(n.prefix) {
			path = path[:len(path)-len(n.prefix)]
		}

		// ルートノードの親ノードはないので nil を返す
		if _n == nil {
			return nil, ""
		}
	}
}

func paramSearch(n *Node, path string) (*Node, string, []*Param) {
	/*
	** パラメータを保持するスライスを定義
	 */
	params := make([]*Param, 0)

	_suffix := path
	now := ""

	for {
		var paramBytes bytes.Buffer
		i := 0
		/*
		** '/' までの文字列を抽出する
		 */
		for ; i < len(_suffix); i++ {
			if _suffix[i] == '/' {
				break
			}
			paramBytes.WriteByte(_suffix[i])
		}

		/*
		** 抽出した文字列をスライスに追加
		 */
		n.param.Value = paramBytes.String()
		params = append(params, &n.param)

		now = _suffix[:i]

		// 既にパスの最後まで処理した場合は、ノードとパスを返す
		if i == len(_suffix) {
			return n, now, params
		}

		// 次のノードは必ず '/' なので、ノードを更新する
		if len(n.children) > 0 {
			n = n.children[0]
			now += "/"
		}

		_suffix = _suffix[lcp(_suffix, now):]
		_next := n

		// パスパタメータの探索は終了
		return _next, now, params
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
