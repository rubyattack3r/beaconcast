#!/bin/sh
# Copies the license files of every module linked into the beaconcast binary,
# on any release platform, into licenses/<module path>/. With --check, only
# reports modules whose licenses are missing or out of date.
set -eu

out=licenses
check=false
[ "${1:-}" = "--check" ] && check=true

modules=$(for os in linux darwin windows; do
	GOOS=$os go list -deps -f '{{with .Module}}{{.Path}}{{end}}' ./cmd/beaconcast
done | sort -u | grep -v '^github.com/rubyattack3r/beaconcast$' | grep .)

status=0
for module in $modules; do
	dir=$(go list -m -f '{{.Dir}}' "$module")
	files=$(cd "$dir" && find . -maxdepth 1 -type f | sed 's|^\./||' | grep -i -E '^(licen[cs]e|copying|notice|patents)' | sort || true)
	if [ -d "$dir/licenses" ]; then
		files="$files $(cd "$dir" && ls licenses/* | grep -i -v 'gpl')"
	fi
	if [ -z "$files" ]; then
		echo "no license file found for $module" >&2
		status=1
		continue
	fi
	for file in $files; do
		target="$out/$module/$file"
		if $check; then
			if ! cmp -s "$dir/$file" "$target"; then
				echo "missing or outdated: $target (run make licenses)" >&2
				status=1
			fi
		else
			mkdir -p "$(dirname "$target")"
			cp "$dir/$file" "$target"
			chmod 644 "$target"
		fi
	done
done
exit $status
