#! /bin/bash

export KUBECONFIG=tmp/kubeconfig.yaml
for i in {0..30}; do
  let count=10*i
  echo "count is: ${count}"
  ./scripts/cleanup.sh
  go run main.go -suite=flagBasedBaseConfig -output-text=incrementalInc${i} -output-csv=incrementalInc${i}.csv -base-gateway-count=$count
  cat incrementalInc${i}.csv >> incrementalInc.csv
  cat incrementalInc${i} >> incrementalInc
done