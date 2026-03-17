package flutter

import (
	"testing"
)

const realSemanticssDump = `SemanticsNode#0
 │ Rect.fromLTRB(0.0, 0.0, 1206.0, 2622.0)
 │
 └─SemanticsNode#1
   │ Rect.fromLTRB(0.0, 0.0, 402.0, 874.0) scaled by 3.0x
   │ textDirection: ltr
   │
   └─SemanticsNode#2
     │ Rect.fromLTRB(0.0, 0.0, 402.0, 874.0)
     │ sortKey: OrdinalSortKey#3d03a(order: 0.0)
     │
     └─SemanticsNode#3
       │ Rect.fromLTRB(0.0, 0.0, 402.0, 874.0)
       │ flags: scopesRoute
       │
       ├─SemanticsNode#6
       │ │ Rect.fromLTRB(0.0, 0.0, 402.0, 118.0)
       │ │
       │ └─SemanticsNode#7
       │     Rect.fromLTRB(86.1, 76.0, 315.9, 104.0)
       │     flags: isHeader
       │     label: "Flutter Demo Home Page"
       │     textDirection: ltr
       │
       ├─SemanticsNode#4
       │   Rect.fromLTRB(47.5, 468.0, 354.5, 488.0)
       │   label: "You have pushed the button this many times:"
       │   textDirection: ltr
       │
       ├─SemanticsNode#5
       │   Rect.fromLTRB(192.5, 488.0, 209.5, 524.0)
       │   label: "0"
       │   textDirection: ltr
       │
       └─SemanticsNode#8
         │ merge boundary ⛔️
         │ Rect.fromLTRB(0.0, 0.0, 56.0, 56.0) with transform
         │ [[1.0,2.4492935982947064e-16,0.0,330.0];
         │ [-2.4492935982947064e-16,1.0,0.0,768.0]; [0.0,0.0,1.0,0.0];
         │ [0.0,0.0,0.0,1.0]]
         │ tooltip: "Increment"
         │ textDirection: ltr
         │
         └─SemanticsNode#9
             merged up ⬆️
             Rect.fromLTRB(0.0, 0.0, 56.0, 56.0)
             actions: tap
             flags: isButton, hasEnabledState, isEnabled, isFocusable
`

func TestParseSemanticsTree(t *testing.T) {
	root := parseSemanticsText(realSemanticssDump)

	if root == nil {
		t.Fatal("root is nil")
	}

	if len(root.Children) == 0 {
		t.Fatal("no children parsed")
	}

	t.Logf("Parsed %d nodes", len(root.Children))

	// Check that labels are found
	labels := root.AllLabels()
	t.Logf("Labels: %v", labels)

	if len(labels) == 0 {
		t.Fatal("no labels found")
	}

	// Find "Increment" (from tooltip)
	node := root.FindByLabel("Increment", 0)
	if node == nil {
		t.Fatalf("could not find 'Increment' node. Available labels: %v", labels)
	}

	t.Logf("Found Increment: ID=%d, Rect=%+v", node.ID, node.Rect)

	if node.Rect == nil {
		t.Fatal("Increment node has no rect")
	}

	// The FAB is at transform offset (330, 768), size 56x56
	// So rect should be approximately (330, 768, 386, 824)
	cx, cy := node.Rect.Center()
	t.Logf("Increment center: (%.1f, %.1f)", cx, cy)

	if cx < 300 || cx > 400 {
		t.Errorf("Increment center X (%.1f) should be around 358", cx)
	}
	if cy < 750 || cy > 830 {
		t.Errorf("Increment center Y (%.1f) should be around 796", cy)
	}
}

func TestFindByLabelFlutterDemo(t *testing.T) {
	root := parseSemanticsText(realSemanticssDump)

	node := root.FindByLabel("Flutter Demo Home Page", 0)
	if node == nil {
		t.Fatal("could not find 'Flutter Demo Home Page'")
	}

	if node.Rect == nil {
		t.Fatal("Flutter Demo Home Page has no rect")
	}

	cx, cy := node.Rect.Center()
	t.Logf("Flutter Demo Home Page center: (%.1f, %.1f)", cx, cy)
}

func TestFindByLabelCounter(t *testing.T) {
	root := parseSemanticsText(realSemanticssDump)

	node := root.FindByLabel("0", 0)
	if node == nil {
		t.Fatal("could not find counter '0'")
	}

	t.Logf("Counter rect: %+v", node.Rect)
}

func TestFindByLabelNotFound(t *testing.T) {
	root := parseSemanticsText(realSemanticssDump)

	node := root.FindByLabel("NonExistent", 0)
	if node != nil {
		t.Error("should not find non-existent label")
	}
}

func TestParseActions(t *testing.T) {
	root := parseSemanticsText(realSemanticssDump)

	// The Increment node should have actions propagated from merged child
	node := root.FindByLabel("Increment", 0)
	if node == nil {
		t.Fatal("could not find Increment")
	}

	t.Logf("Increment actions: %v, flags: %v", node.Actions, node.Flags)

	if len(node.Actions) == 0 {
		t.Error("Increment should have 'tap' action (propagated from merged child)")
	}
}
