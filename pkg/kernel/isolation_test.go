package kernel

import (
	"testing"
)

func TestProcessMemoryIsolation(t *testing.T) {
	lab := NewMemoryIsolationLab()
	res := lab.RunIsolationTest()

	if res.ProcA_Value != "Secret_Token_A" {
		t.Errorf("Expected Process A value 'Secret_Token_A', got '%s'", res.ProcA_Value)
	}

	if res.ProcB_Value != "Public_Data_B" {
		t.Errorf("Expected Process B value 'Public_Data_B', got '%s'", res.ProcB_Value)
	}

	if res.ProcA_Physical == res.ProcB_Physical {
		t.Errorf("Expected different physical addresses for isolated processes, got 0x%X", res.ProcA_Physical)
	}

	if !res.CrossAccessTrap {
		t.Errorf("Expected illegal cross-access to trigger hardware trap")
	}

	report := RenderIsolationReport(res)
	if len(report) == 0 {
		t.Errorf("Expected non-empty isolation report")
	}
}
