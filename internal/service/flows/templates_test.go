package flows

import "testing"

// Every built-in template must be a valid IVR spec whose happy path (press "1")
// reaches a terminal outcome — otherwise the builder would ship a broken starter.
func TestTemplatesAreValidAndSimulate(t *testing.T) {
	svc := New(nil)
	tpls := Templates()
	if len(tpls) == 0 {
		t.Fatal("expected built-in templates")
	}
	for _, tpl := range tpls {
		t.Run(tpl.Name, func(t *testing.T) {
			if err := validateIVR(tpl.Spec); err != nil {
				t.Fatalf("template %q has an invalid spec: %v", tpl.Name, err)
			}
			sim, err := svc.Simulate(tpl.Spec, []string{"1"})
			if err != nil {
				t.Fatalf("simulate %q: %v", tpl.Name, err)
			}
			if !sim.Ended || sim.Result == "" {
				t.Errorf("template %q happy path did not reach an outcome: %+v", tpl.Name, sim)
			}
		})
	}
}
