module github.com/iden3/go-circom-witnesscalc

go 1.14

require (
	github.com/iden3/go-wasm3 v0.0.0-20200407092348-656263e6984f
	github.com/sirupsen/logrus v1.5.0
	github.com/stretchr/testify v1.5.1
)

replace github.com/iden3/go-wasm3 => ../new/go-wasm3
