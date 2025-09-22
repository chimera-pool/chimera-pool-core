#!/bin/bash

if [[ $# -ne 1 ]]; then
    echo "Usage: $0 <feature-name>"
    echo "Example: $0 advanced-mining-analytics"
    exit 1
fi

FEATURE_NAME="$1"
SPEC_NUMBER=$(printf "%03d" $(($(ls specs/ | grep -E '^[0-9]{3}-' | wc -l) + 1)))
SPEC_DIR="specs/${SPEC_NUMBER}-${FEATURE_NAME}"

echo "Creating new feature specification: ${FEATURE_NAME}"
echo "Spec directory: ${SPEC_DIR}"

mkdir -p "${SPEC_DIR}"/{contracts,data-model}

sed "s/\[FEATURE_NAME\]/${FEATURE_NAME}/g" templates/spec-template.md > "${SPEC_DIR}/spec.md"
sed "s/\[FEATURE_NAME\]/${FEATURE_NAME}/g" templates/plan-template.md > "${SPEC_DIR}/plan.md"

touch "${SPEC_DIR}/research.md"
touch "${SPEC_DIR}/quickstart.md"

echo "✅ Feature specification created successfully!"
echo "Next steps:"
echo "1. Edit ${SPEC_DIR}/spec.md to define the feature requirements"
echo "2. Use Claude to help develop the specification and implementation plan"
