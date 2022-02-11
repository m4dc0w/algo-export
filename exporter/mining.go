package exporter

import (
	"fmt"
)

// MiningPlanets exports transactions as an airdrop based on certain criteria.
func MiningPlanets(records []ExportRecord) ([]ExportRecord, error) {
	// Assume Mining transactions are usually done in 1 ASA deposit transaction.
	if !IsLengthExcludeReward(records, 1) || !records[0].IsASAMining(27165954) {
		return records, fmt.Errorf("invalid MiningPlanets() record")
	}

	r := records[0]

	// https://explorer.planetwatch.io/
	// PlanetWatch mining reward address ZW3ISEHZUHPO7OZGMKLKIIMKVICOUDRCERI454I3DB2BH52HGLSO67W754.
	// ASA 27165954 (PLANET)
	if r.sender == "ZW3ISEHZUHPO7OZGMKLKIIMKVICOUDRCERI454I3DB2BH52HGLSO67W754" && r.sender != r.account {
		records[0].mining = true
	}
	return records, nil
}
