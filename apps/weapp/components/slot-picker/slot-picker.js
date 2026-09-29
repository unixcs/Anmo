// slot-picker — 上下午必选 + 可选具体时间（D20）。
// 输入 booking-options 的 {open, am, pm}；选中变化时 triggerEvent('change', {part, slot})。
// 模糊预约（只选上下午）与具体时间共享名额池：slots.remaining<1 的槽禁选。
// part/slot 一律由父页面持有（受控组件）：换日期、提交成功、改期重置后高亮必然同步。
Component({
  properties: {
    options: { type: Object, value: null }, // BookingOptions
    part: { type: String, value: '' }, // 'AM' | 'PM' | ''
    slot: { type: String, value: '' }, // 'HH:MM' | ''
  },

  methods: {
    pickPart(e) {
      const part = e.currentTarget.dataset.part
      const half = part === 'AM' ? this.data.options.am : this.data.options.pm
      // 闭店、当日时段已过、半日池已空都不给选
      if (!half || half.closed || half.ended || Number(half.remaining) < 1) return
      this.triggerEvent('change', { part, slot: '' })
    },

    // 点具体时间即隐含选定所在半天（父页面 part 可能还没选）
    pickSlot(e) {
      const { time, remaining, part } = e.currentTarget.dataset
      if (Number(remaining) < 1) return // 满槽不可点
      this.triggerEvent('change', { part, slot: time })
    },
  },
})
