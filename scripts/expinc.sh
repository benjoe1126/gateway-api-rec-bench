#! /bin/bash

export KUBECONFIG=tmp/kubeconfig.yaml
for i in {1..10}; do
  let count=2**i
  echo "count is: ${count}"
  ./scripts/cleanup.sh
  go run main.go -suite=flagBasedBaseConfig -output-text=expInc${i} -output-csv=expInc${i}.csv -base-gateway-count=$count
  cat expInc${i}.csv >> expInc.csv
  cat expInc${i} >> expInc
done