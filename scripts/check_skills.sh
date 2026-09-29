#!/usr/bin/env bash
# skills 契约校验：SKILL.md frontmatter（name/description）与 reference 引用完整性。
#
# 逐 skill 断言：
#   ① skills/<name>/SKILL.md 存在，且以 YAML frontmatter（首行 --- 起）开头
#   ② frontmatter 的 name 等于目录名，且匹配 ^[a-z0-9]+(-[a-z0-9]+)*$
#   ③ description 存在且非空
#   ④ SKILL.md 里每个 references/<file>.md 引用都指向存在的文件
#   ⑤ references/*.md 无孤儿（每个都被 SKILL.md 引用）
#   ⑥ skills/ 下不存在没有 SKILL.md 的目录；且至少有一个合法 skill
#   ⑦ SKILL.md 与 references/*.md 里的每个 markdown 相对链接都能解析到磁盘上的文件
#      （http(s)/mailto/纯锚点除外）—— 实现期真踩过：SKILL.md 位于 skills/<name>/，
#      上溯仓库根是 ../../ 而不是 ../../../，写错会静默跳出仓库且旧断言看不见
#
# 接入范围：只作 `make check-skills`，不加入 make ci / make release-check
#（设计 §6：聚合目标的发布语义不变）。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SKILLS_DIR="$ROOT/skills"

fail=0
err() { printf '❌ %s\n' "$1" >&2; fail=1; }

[ -d "$SKILLS_DIR" ] || { echo "❌ 缺少目录 skills/" >&2; exit 1; }

count=0
for dir in "$SKILLS_DIR"/*/; do
	[ -d "$dir" ] || continue
	name="$(basename "$dir")"
	entry="$dir/SKILL.md"

	if [ ! -f "$entry" ]; then
		err "skills/$name 缺少 SKILL.md"
		continue
	fi
	count=$((count + 1))

	# frontmatter：首个 --- 与下一个 --- 之间的块
	fm="$(awk 'NR==1 && $0=="---" {inside=1; next} inside && $0=="---" {exit} inside {print}' "$entry")"
	if [ -z "$fm" ]; then
		err "skills/$name/SKILL.md 缺少 YAML frontmatter（首行必须是 ---）"
	fi

	fmname="$(printf '%s\n' "$fm" | sed -n 's/^name:[[:space:]]*//p' | head -1)"
	fmdesc="$(printf '%s\n' "$fm" | sed -n 's/^description:[[:space:]]*//p' | head -1)"

	if [ "$fmname" != "$name" ]; then
		err "skills/$name: frontmatter name='$fmname' 与目录名不一致"
	fi
	if ! printf '%s' "$fmname" | grep -Eq '^[a-z0-9]+(-[a-z0-9]+)*$'; then
		err "skills/$name: name 必须匹配 ^[a-z0-9]+(-[a-z0-9]+)*$"
	fi
	if [ -z "${fmdesc//[[:space:]]/}" ]; then
		err "skills/$name: description 为空"
	fi

	# ④ SKILL.md 提到的 reference 必须存在
	while IFS= read -r ref; do
		[ -n "$ref" ] || continue
		[ -f "$dir/$ref" ] || err "skills/$name: 引用了不存在的 $ref"
	done < <(grep -oE 'references/[A-Za-z0-9._-]+\.md' "$entry" | sort -u)

	# ⑤ references/ 下不得有孤儿
	if [ -d "$dir/references" ]; then
		for f in "$dir"/references/*.md; do
			[ -f "$f" ] || continue
			rel="references/$(basename "$f")"
			grep -qF "$rel" "$entry" || err "skills/$name: $rel 未被 SKILL.md 引用（孤儿）"
		done
	fi

	# ⑦ 相对链接可解析（SKILL.md 与每份 reference；跳过 url / mailto / 纯锚点）
	for md in "$entry" "$dir"/references/*.md; do
		[ -f "$md" ] || continue
		md_dir="$(dirname "$md")"
		while IFS= read -r target; do
			[ -n "$target" ] || continue
			case "$target" in
				http://*|https://*|mailto:*|'#'*) continue ;;
			esac
			path="${target%%#*}"
			[ -n "$path" ] || continue
			[ -e "$md_dir/$path" ] || err "skills/$name: $(basename "$md") 的相对链接无法解析：$target"
		done < <(grep -oE '\]\([^)]+\)' "$md" | sed -e 's/^](//' -e 's/)$//')
	done
done

if [ "$count" -eq 0 ]; then
	err "skills/ 下没有合法的 skill 目录"
fi
if [ "$fail" -ne 0 ]; then
	echo "❌ check-skills: 见上方错误（修法：frontmatter 的 name 与目录同名、description 非空、引用与文件一一对应、相对链接按所在文件的上溯层数写对）" >&2
	exit 1
fi
echo "✅ check-skills: $count 个 skill 契约完整"
