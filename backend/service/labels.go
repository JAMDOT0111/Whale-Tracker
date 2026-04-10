package service

import "strings"

type AddressLabel struct {
	Tag  string // "exchange", "bridge", "defi"
	Name string // e.g. "Binance", "Wormhole"
}

var knownAddresses = map[string]AddressLabel{
	// ===== Exchanges =====

	// Binance
	"0x28c6c06298d514db089934071355e5743bf21d60": {Tag: "exchange", Name: "Binance"},
	"0x21a31ee1afc51d94c2efccaa2092ad1028285549": {Tag: "exchange", Name: "Binance"},
	"0xdfd5293d8e347dfe59e90efd55b2956a1343963d": {Tag: "exchange", Name: "Binance"},
	"0x56eddb7aa87536c09ccc2793473599fd21a8b17f": {Tag: "exchange", Name: "Binance"},
	"0xf977814e90da44bfa03b6295a0616a897441acec": {Tag: "exchange", Name: "Binance"},
	"0x631fc1ea2270e98fbd9d92658ece0f5a269aa161": {Tag: "exchange", Name: "Binance"},
	"0xbe0eb53f46cd790cd13851d5eff43d12404d33e8": {Tag: "exchange", Name: "Binance"},
	"0x5a52e96bacdabb82fd05763e25335261b270efcb": {Tag: "exchange", Name: "Binance"},

	// Coinbase
	"0x71660c4005ba85c37ccec55d0c4493e66fe775d3": {Tag: "exchange", Name: "Coinbase"},
	"0x503828976d22510aad0201ac7ec88293211d23da": {Tag: "exchange", Name: "Coinbase"},
	"0xddfabcdc4d8ffc6d5beaf154f18b778f892a0740": {Tag: "exchange", Name: "Coinbase"},
	"0x3cd751e6b0078be393132286c442345e68ff0afc": {Tag: "exchange", Name: "Coinbase"},
	"0xb5d85cbf7cb3ee0d56b3bb207d5fc4b82f43f511": {Tag: "exchange", Name: "Coinbase"},
	"0xa9d1e08c7793af67e9d92fe308d5697fb81d3e43": {Tag: "exchange", Name: "Coinbase"},
	"0x02466e547bfdab679fc49e96bbfc62b9747d997c": {Tag: "exchange", Name: "Coinbase"},

	// Kraken
	"0x2910543af39aba0cd09dbb2d50200b3e800a63d2": {Tag: "exchange", Name: "Kraken"},
	"0x05ff6964d21e5dae3b1010d5ae0465b3c450f381": {Tag: "exchange", Name: "Kraken"},
	"0xda9dfa130df4de4673b89022ee50ff26f6ea73cf": {Tag: "exchange", Name: "Kraken"},
	"0x267be1c1d684f78cb4f6a176c4911b741e4ffdc0": {Tag: "exchange", Name: "Kraken"},

	// OKX
	"0x6cc5f688a315f3dc28a7781717a9a798a59fda7b": {Tag: "exchange", Name: "OKX"},
	"0x236f9f97e0e62388479bf9e5ba4889e46b0273c3": {Tag: "exchange", Name: "OKX"},
	"0xa7efae728d2936e78bda97dc267687568dd593f3": {Tag: "exchange", Name: "OKX"},
	"0x559432e18b281731c054cd703d4b49872be4ed53": {Tag: "exchange", Name: "OKX"},

	// Bybit
	"0xf89d7b9c864f589bbf53a82105107622b35eaa40": {Tag: "exchange", Name: "Bybit"},
	"0xee5b5b923ffce93a870b3104b7ca09c3db80047a": {Tag: "exchange", Name: "Bybit"},
	"0xa4b9569bf942c3aad23c0c2d322fe4aff8e1bf30": {Tag: "exchange", Name: "Bybit"},

	// Bitfinex
	"0x77134cbc06cb00b66f4c7e623d5fdbf6777635ec": {Tag: "exchange", Name: "Bitfinex"},
	"0x1151314c646ce4e0efd76d1af4760ae66a9fe30f": {Tag: "exchange", Name: "Bitfinex"},

	// Gemini
	"0xd24400ae8bfebb18ca49be86258a3c749cf46853": {Tag: "exchange", Name: "Gemini"},
	"0x6fc82a5fe25a5cdb58bc74600a40a69c065263f8": {Tag: "exchange", Name: "Gemini"},

	// KuCoin
	"0xf16e9b0d03470827a95cdfd0cb8a8a3b46969b91": {Tag: "exchange", Name: "KuCoin"},
	"0xd6216fc19db775df9774a6e33526131da7d19a2c": {Tag: "exchange", Name: "KuCoin"},

	// Gate.io
	"0x0d0707963952f2fba59dd06f2b425ace40b492fe": {Tag: "exchange", Name: "Gate.io"},
	"0x1c4b70a3968436b9a0a9cf5205c787eb81bb558c": {Tag: "exchange", Name: "Gate.io"},

	// HTX (Huobi)
	"0xab5c66752a9e8167967685f1450532fb96d5d24f": {Tag: "exchange", Name: "HTX"},
	"0x6748f50f686bfbca6fe8ad62b22228b87f31ff2b": {Tag: "exchange", Name: "HTX"},
	"0xfdb16996831753d5331ff813c29a93c76834a0ad": {Tag: "exchange", Name: "HTX"},

	// MEXC
	"0x75e89d5979e4f6fba9f97c104c2f0afb3f1dcb88": {Tag: "exchange", Name: "MEXC"},
	"0x9642b23ed1e01df1092b92641051881a322f5d4e": {Tag: "exchange", Name: "MEXC"},

	// Coincheck
	"0x8696e84ab5e78983f2456bcb5c199eea9648c8c2": {Tag: "exchange", Name: "Coincheck"},

	// QuadrigaCX
	"0x1e143b2588705dfea63a17f2032ca123df995ce0": {Tag: "exchange", Name: "QuadrigaCX"},
	"0x5b5b69f4e0add2df5d2176d7dbd20b4897bc7ec4": {Tag: "exchange", Name: "QuadrigaCX"},

	// BtcTurk
	"0x76ec5a0d3632b2133d9f1980903305b62678fbd3": {Tag: "exchange", Name: "BtcTurk"},

	// Bitpanda
	"0xb10edd6fa6067dba8d4326f1c8f0d1c791594f13": {Tag: "exchange", Name: "Bitpanda"},
	"0xf197c6f2ac14d25ee2789a73e4847732c7f16bc9": {Tag: "exchange", Name: "Bitpanda"},

	// bitFlyer
	"0x111cff45948819988857bbf1966a0399e0d1141e": {Tag: "exchange", Name: "bitFlyer"},

	// Bullish
	"0x100ae042ef0ea159ecc3513e9a378ff21f3829ba": {Tag: "exchange", Name: "Bullish"},

	// HashKey
	"0xee1bf4d7c53af2beafc7dc1dcea222a8c6d87ad9": {Tag: "exchange", Name: "HashKey Exchange"},

	// Bitso
	"0xe3ecd65cf2ad2eba2aa2be1d0894753b2172abd1": {Tag: "exchange", Name: "Bitso"},

	// eToro
	"0x77fb357f55bef5a70d30663955f8c9f35794df0e": {Tag: "exchange", Name: "eToro"},
	"0x434587332cc35d33db75b93f4f27cc496c67a4db": {Tag: "exchange", Name: "eToro"},

	// Coinone
	"0x1e2fcfd26d36183f1a5d90f0e6296915b02bcb40": {Tag: "exchange", Name: "Coinone"},

	// Coinhako
	"0xf4e6deea1b4da85c2d68db8d771d37ec1148b853": {Tag: "exchange", Name: "Coinhako"},

	// Bitget
	"0x59708733fbbf64378d9293ec56b977c011a08fd2": {Tag: "exchange", Name: "Bitget"},

	// ===== DEX / DeFi =====

	// Uniswap
	"0xe592427a0aece92de3edee1f18e0157c05861564": {Tag: "defi", Name: "Uniswap V3 Router"},
	"0x68b3465833fb72a70ecdf485e0e4c7bd8665fc45": {Tag: "defi", Name: "Uniswap Universal Router"},
	"0x7a250d5630b4cf539739df2c5dacb4c659f2488d": {Tag: "defi", Name: "Uniswap V2 Router"},

	// 1inch
	"0x1111111254eeb25477b68fb85ed929f73a960582": {Tag: "defi", Name: "1inch Router"},
	"0x111111125421ca6dc452d289314280a0f8842a65": {Tag: "defi", Name: "1inch Aggregation Router"},

	// Aave
	"0x87870bca3f3fd6335c3f4ce8392d69350b4fa4e2": {Tag: "defi", Name: "Aave V3 Pool"},

	// Curve
	"0x16c6521dff6bab339122a0fe25a9116693265353": {Tag: "defi", Name: "Curve Router"},

	// Maker
	"0xc3d688b66703497daa19211eedff47f25384cdc3": {Tag: "defi", Name: "Maker DSR Manager"},

	// ===== Cross-Chain Bridges =====

	// Wormhole
	"0x98f3c9e6e3face36baad05fe09d375ef1464288b": {Tag: "bridge", Name: "Wormhole"},
	"0x3ee18b2214aff97000d974cf647e7c347e8fa585": {Tag: "bridge", Name: "Wormhole"},

	// Stargate / LayerZero
	"0x296f55f8fb28e498b858d0bcda06d955b2cb3f97": {Tag: "bridge", Name: "Stargate"},
	"0x66a71dcef29a0ffbdbe3c6a460a3b5bc225cd675": {Tag: "bridge", Name: "LayerZero"},

	// Across Protocol
	"0x5c7bcd6e7de5423a257d81b442095a1a6ced35c5": {Tag: "bridge", Name: "Across"},
	"0x4d9079bb4165aeb4084c526a32695dcfd2f77381": {Tag: "bridge", Name: "Across"},

	// Hop Protocol
	"0xb8901acb165ed027e32754e0ffe830802919727f": {Tag: "bridge", Name: "Hop"},
	"0x3666f603cc164936c1b87e207f36beba4ac5f18a": {Tag: "bridge", Name: "Hop"},

	// Celer cBridge
	"0x5427fefa711eff984124bfbb1ab6fbf5e3da1820": {Tag: "bridge", Name: "Celer cBridge"},

	// Synapse
	"0x2796317b0ff8538f253012862c06787adfb8ceb6": {Tag: "bridge", Name: "Synapse"},
	"0x6571d6be3d8460cf5f7d6711cd9961860029d85f": {Tag: "bridge", Name: "Synapse"},

	// Arbitrum Bridge
	"0x8315177ab297ba92a06054ce80a67ed4dbd7ed3a": {Tag: "bridge", Name: "Arbitrum Bridge"},
	"0x011b6e24ffb0b5f5fcc564cf4183c5bbbc96d515": {Tag: "bridge", Name: "Arbitrum Bridge"},

	// Optimism Bridge
	"0x99c9fc46f92e8a1c0dec1b1747d010903e884be1": {Tag: "bridge", Name: "Optimism Bridge"},
	"0x467194771dae2967aef3ecbedd3bf9a310c76c65": {Tag: "bridge", Name: "Optimism Bridge"},

	// Polygon Bridge
	"0xa0c68c638235ee32657e8f720a23cec1bfc77c77": {Tag: "bridge", Name: "Polygon Bridge"},
	"0x40ec5b33f54e0e8a33a975908c5ba1c14e5bbbdf": {Tag: "bridge", Name: "Polygon Bridge"},

	// Base Bridge
	"0x3154cf16ccdb4c6d922629664174b904d80f2c35": {Tag: "bridge", Name: "Base Bridge"},
	"0x49048044d57e1c92a77f79988d21fa8faf74e97e": {Tag: "bridge", Name: "Base Bridge"},

	// zkSync Era Bridge
	"0x32400084d98056463ffb6765a0ad14ef1ad25f45": {Tag: "bridge", Name: "zkSync Bridge"},
}

func LookupAddress(address string) *AddressLabel {
	label, ok := knownAddresses[strings.ToLower(address)]
	if !ok {
		return nil
	}
	return &label
}
