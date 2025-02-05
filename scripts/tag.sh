
version=$(go run ./cmd/secrets -version | awk '{ print $2 }' | awk -F= '{ print $2 }')

git tag v${version}
git tag chart-${version}
