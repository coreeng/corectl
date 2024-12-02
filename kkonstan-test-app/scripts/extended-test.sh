#!/bin/bash
set -euo pipefail

subenv=$1
tenant_name=$2
app_name=${3:-$tenant_name}
scale_down=${4:-false}
timeout=${5:-"15m"}
test_name=${app_name}-${subenv}-test

./scripts/helm-test.sh $subenv $tenant_name $app_name false $timeout $test_name

kubectl wait --for=jsonpath='{.status.stage}'=finished testrun/${test_name} -n ${namespace} --timeout ${timeout}

./scripts/helm-test.sh $subenv $tenant_name $app_name $scale_down $timeout $test_name-validate
