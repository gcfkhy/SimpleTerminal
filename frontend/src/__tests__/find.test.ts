import { describe, it, expect } from 'vitest'
import { findMatches, locateOffset, buildTextIndex, MAX_MATCHES, type TextSeg } from '../utils/markdown/find'

describe('findMatches', () => {
  it('不区分大小写（快路径）找出全部不重叠匹配', () => {
    const m = findMatches('Hello hello HELLO', 'hello', false)
    expect(m.map((x) => x.start)).toEqual([0, 6, 12])
    expect(m[0]).toEqual({ start: 0, end: 5 })
  })

  it('区分大小写只匹配同形', () => {
    const m = findMatches('Hello hello', 'hello', true)
    expect(m).toEqual([{ start: 6, end: 11 }])
  })

  it('空串/空查询返回空', () => {
    expect(findMatches('', 'x', false)).toEqual([])
    expect(findMatches('abc', '', false)).toEqual([])
  })

  it('Unicode 折叠改变长度时走慢路径，命中映射回原串偏移', () => {
    // 'İ'(U+0130).toLowerCase() 长度变 2，触发慢路径
    const m = findMatches('aİb', 'b', false)
    expect(m).toEqual([{ start: 2, end: 3 }])
  })

  it('命中数达 MAX_MATCHES 截断', () => {
    const m = findMatches('a'.repeat(MAX_MATCHES + 1000), 'a', true)
    expect(m.length).toBe(MAX_MATCHES)
  })
})

describe('locateOffset', () => {
  function segOf(text: string, start: number): TextSeg {
    return { node: document.createTextNode(text), start }
  }
  const segs = [segOf('abc', 0), segOf('def', 3)] // 拼接 "abcdef"

  it('起点落在节点内', () => {
    expect(locateOffset(segs, 2, false)).toEqual({ node: segs[0].node, offset: 2 })
  })

  it('起点落在边界归右侧节点开头', () => {
    expect(locateOffset(segs, 3, false)).toEqual({ node: segs[1].node, offset: 0 })
  })

  it('终点落在边界归左侧节点末尾', () => {
    expect(locateOffset(segs, 3, true)).toEqual({ node: segs[0].node, offset: 3 })
  })

  it('空段返回 null', () => {
    expect(locateOffset([], 0, false)).toBeNull()
  })
})

describe('buildTextIndex', () => {
  it('拼接文本并跳过 .katex-mathml', () => {
    const body = document.createElement('div')
    body.innerHTML = '<p>foo</p><span class="katex-mathml">HIDDEN</span><p>bar</p>'
    const idx = buildTextIndex(body)
    expect(idx.full).toBe('foobar')
    expect(idx.segs.length).toBe(2)
    expect(idx.segs[1].start).toBe(3)
  })

  it('bodyEl 为 null 返回空索引', () => {
    expect(buildTextIndex(null)).toEqual({ full: '', segs: [] })
  })
})
