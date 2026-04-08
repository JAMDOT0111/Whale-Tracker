package service

import (
"eth-sweeper/model"
)

// GetAllTxsForPrediction bypasses graph limits and gets up to 10k items per type for ML context
func (c *EtherscanClient) GetAllTxsForPrediction(address string) (normal, internal, token []model.Transaction, err error) {
normal, _ = c.fetchNormalTx(address, 1, 10000)
internal, _ = c.fetchInternalTx(address, 1, 10000)
token, _ = c.fetchTokenTx(address, 1, 10000)
return normal, internal, token, nil
}
