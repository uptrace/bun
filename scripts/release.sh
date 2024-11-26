#!/bin/bash

set -e

help() {
    cat <<- EOF
Usage: TAG=tag $0

Updates version in go.mod files and pushes a new branch to GitHub.

VARIABLES:
  TAG        git tag, for example, v1.0.0
EOF
    exit 0
}

if [ -z "$TAG" ]
then
    printf "TAG is required\n\n"
    help
fi

TAG_REGEX="^v(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(\\-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?(\\+[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"
if ! [[ "${TAG}" =~ ${TAG_REGEX} ]]; then
    printf "TAG is not valid: ${TAG}\n\n"
    exit 1
fi

TAG_FOUND=`git tag --list ${TAG}`
if [[ ${TAG_FOUND} = ${TAG} ]] ; then
    printf "tag ${TAG} already exists\n\n"
    exit 1
fi

if ! git diff --quiet
then
    printf "working tree is not clean\n\n"
    git status
    exit 1
fi

git checkout master
make go_mod_tidy

PACKAGE_DIRS=$(find . -mindepth 2 -type f -name 'go.mod' -exec dirname {} \; \
  | sed 's/^\.\///' \
  | sort)

if [[ "$(uname)" == "Darwin" ]]; then
    for dir in $PACKAGE_DIRS
    do
        sed -i "" \
          "s/uptrace\/bun\([^ ]*\) v.*/uptrace\/bun\1 ${TAG}/" "${dir}/go.mod"
    done

    for file in $(find . -type f -name 'version.go')
    do
        sed -i "" "/func Version() string/{n;s/\(return \)\"[^\"]*\"/\1\"${TAG#v}\"/;}" ${file}
    done
    sed -i "" "s/\(\"version\": \)\"[^\"]*\"/\1\"${TAG#v}\"/" ./package.json

else
    for dir in $PACKAGE_DIRS
    do
        sed --in-place \
          "s/uptrace\/bun\([^ ]*\) v.*/uptrace\/bun\1 ${TAG}/" "${dir}/go.mod"
    done

    for file in $(find . -type f -name 'version.go')
    do
        sed --in-place "/func Version() string/{n;s/\(return \)\"[^\"]*\"/\1\"${TAG#v}\"/;}" ${file}
    done
    sed --in-place "s/\(\"version\": \)\"[^\"]*\"/\1\"${TAG#v}\"/" ./package.json
fi

conventional-changelog -p angular -i CHANGELOG.md -s

git checkout -b release/${TAG} master
git add -u
git commit -m "chore: release $TAG (release.sh)"
git push origin release/${TAG}
