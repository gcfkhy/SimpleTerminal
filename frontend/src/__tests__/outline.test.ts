import { describe, it, expect } from 'vitest'
import { extractHeadings, buildOutlineTree, type OutlineItem } from '../utils/markdown/outline'

function h(level: number, text: string, id: string): OutlineItem {
  const el = document.createElement('h' + level)
  el.id = id
  el.textContent = text
  return { level, text, id, el }
}

describe('buildOutlineTree', () => {
  it('常规层级构建嵌套树', () => {
    const tree = buildOutlineTree([h(1, 'A', 'a'), h(2, 'B', 'b'), h(2, 'C', 'c'), h(3, 'D', 'd')])
    expect(tree.length).toBe(1)
    expect(tree[0].id).toBe('a')
    expect(tree[0].children.map((n) => n.id)).toEqual(['b', 'c'])
    expect(tree[0].children[1].children.map((n) => n.id)).toEqual(['d'])
  })

  it('跳级 h1→h3 时 h3 仍挂到最近的更浅标题下', () => {
    const tree = buildOutlineTree([h(1, 'A', 'a'), h(3, 'C', 'c')])
    expect(tree.length).toBe(1)
    expect(tree[0].children.map((n) => n.id)).toEqual(['c'])
  })

  it('空输入返回空数组', () => {
    expect(buildOutlineTree([])).toEqual([])
  })
})

describe('extractHeadings', () => {
  it('抠出 h1..h6 并排除无 id 或空文本的标题', () => {
    const body = document.createElement('div')
    body.innerHTML =
      '<h1 id="a">A</h1>' +
      '<h2>无 id</h2>' +          // 排除
      '<h2 id="blank">   </h2>' + // 排除（空文本）
      '<h3 id="b">B</h3>'
    const items = extractHeadings(body)
    expect(items.map((i) => i.id)).toEqual(['a', 'b'])
    expect(items[0].level).toBe(1)
    expect(items[1].level).toBe(3)
  })

  it('bodyEl 为 null 返回空数组', () => {
    expect(extractHeadings(null)).toEqual([])
  })
})
