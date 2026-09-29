#!/usr/bin/env bash
# 把仓库内 skills/<name>/ 软链到各 Agent 的项目级发现路径。
#
# 事实源：skills/<name>/（入库）
# 目标：  .claude/skills/<name>（Claude Code 项目级）
#         .agents/skills/<name>（本工作区既有的 Agent skills 路径）
# 两者都在 .gitignore 内（本仓「AI 工具目录不入库」的既有约定）。
#
# 冲突策略：目标已存在且不是指向本仓 skills/<name> 的软链（真实目录/文件，或外来软链）
#           → 打印冲突并非零退出，**绝不覆盖**（保护 .agents/skills 里的第三方 skill 包）。
# 幂等：已指向本仓同一事实源 → 打印 unchanged 并跳过。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SKILLS_DIR="$ROOT/skills"
TARGETS=(".claude/skills" ".agents/skills")

[ -d "$SKILLS_DIR" ] || { echo "❌ 缺少目录 skills/" >&2; exit 1; }

changed=0
found=0
for dir in "$SKILLS_DIR"/*/; do
	[ -f "$dir/SKILL.md" ] || continue
	name="$(basename "$dir")"
	src="${dir%/}"
	found=$((found + 1))

	for rel in "${TARGETS[@]}"; do
		dst_dir="$ROOT/$rel"
		dst="$dst_dir/$name"
		mkdir -p "$dst_dir"

		if [ -L "$dst" ]; then
			current="$(readlink "$dst")"
			if [ "$current" = "$src" ]; then
				printf 'unchanged %s/%s\n' "$rel" "$name"
				continue
			fi
			echo "❌ $rel/$name 已是指向 $current 的软链，拒绝覆盖（手工处理后再跑）" >&2
			exit 1
		fi
		if [ -e "$dst" ]; then
			echo "❌ $rel/$name 已存在且不是软链，拒绝覆盖（第三方 skill 包请保留）" >&2
			exit 1
		fi

		ln -sfn "$src" "$dst"
		printf 'installed %s/%s -> %s\n' "$rel" "$name" "$src"
		changed=$((changed + 1))
	done
done

if [ "$found" -eq 0 ]; then
	echo "❌ skills/ 下没有带 SKILL.md 的 skill" >&2
	exit 1
fi
if [ "$changed" -eq 0 ]; then
	echo "no changes（全部已安装）"
fi
echo "重启 Agent 会话即生效（事实源：skills/）"
