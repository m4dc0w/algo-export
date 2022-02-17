package exporter

// Verified ASA that export well to tax tracker websites (i.e. cointracking.info).
// The string value allows for token disambiguation to the tax software.
// Token disambiguation is needed when there are multiple coins with the same name.
// https://cointracking.info/coin_charts.php
var verifiedASA = map[uint64]string{
	163650: "",         // ARCC
	31566704: "",       // USDC
	137594422: "",      // HDL
	226701642: "",      // YLDY
	230946361: "GEMS3", // Algogems
	27165954: "",       // PLANETS
	283820866: "",      // XET
	287867876: "",      // OPUL
}
