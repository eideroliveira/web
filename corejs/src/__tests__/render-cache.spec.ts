import { mount } from '@vue/test-utils'
import { describe, it, expect, beforeEach } from 'vitest'
import { componentByTemplate, __renderCacheForTest } from '@/utils'

// These guard the fix for the translation-panel memory leak: go-plaid repaints
// portals on a timer with templates that bake in wall-clock text, so every tick
// is a fresh template string. Vue's built-in runtime compiler caches those
// forever; componentByTemplate compiles through a bounded LRU instead.
describe('componentByTemplate bounded compile cache', () => {
  beforeEach(() => {
    __renderCacheForTest.clear()
  })

  it('renders a compiled template', () => {
    const Comp = componentByTemplate(`<div class="hello">{{ locals.name }}</div>`, {}, {
      name: 'world'
    })
    const wrapper = mount(Comp)
    expect(wrapper.find('.hello').text()).toBe('world')
  })

  it('never grows past the cap, even with far more distinct templates', () => {
    const distinct = __renderCacheForTest.max + 200
    for (let i = 0; i < distinct; i++) {
      // Each iteration is a unique template string, mimicking a panel whose
      // baked-in elapsed time ticks up on every poll.
      const Comp = componentByTemplate(`<div class="tick">elapsed ${i}s</div>`, {})
      const wrapper = mount(Comp)
      expect(wrapper.find('.tick').text()).toBe(`elapsed ${i}s`)
      wrapper.unmount()
    }
    expect(__renderCacheForTest.size()).toBeLessThanOrEqual(__renderCacheForTest.max)
  })

  it('reuses the cached render for an identical template', () => {
    const tmpl = `<div class="same">stable</div>`
    componentByTemplate(tmpl, {})
    expect(__renderCacheForTest.size()).toBe(1)
    // An idle poll re-sending the exact same HTML must not add a new entry.
    componentByTemplate(tmpl, {})
    expect(__renderCacheForTest.size()).toBe(1)
  })

  it('recompiles correctly after an entry is evicted', () => {
    const first = `<div class="evictme">first ${0}</div>`
    componentByTemplate(first, {})
    // Overflow the cache so the earliest entry is evicted.
    for (let i = 1; i <= __renderCacheForTest.max; i++) {
      componentByTemplate(`<div class="evictme">first ${i}</div>`, {})
    }
    // The evicted template must still render when it reappears.
    const Comp = componentByTemplate(first, {})
    const wrapper = mount(Comp)
    expect(wrapper.find('.evictme').text()).toBe('first 0')
  })
})
