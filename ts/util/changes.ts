/**
 * Subscribable properties container.
 */

type HookHandler = (arg: any) => void
type HookMap = { [key: string]: HookHandler[] }

export interface ChangeEmitter {
  onChange(key: string, func: HookHandler): void
}

// Wrap an object with a Proxy that executes handlers on property changes.
// To add new handlers, call the .onChange method on the object.
// For type safety, the passed generic interface must extend ChangeEmitter.
export function emitChanges<T extends ChangeEmitter>(obj: T): T {
  const changeHooks: HookMap = {}

  // Proxies do not have a prototype. Some hacks required.
  // Add a function to be executed, when a key is set on the object.
  obj.onChange = (key: string, func: HookHandler) => {
    const hooks = changeHooks[key]
    if (hooks) {
      hooks.push(func)
    } else {
      changeHooks[key] = [func]
    }
  }

  return new Proxy<T>(obj, {
    set(target: T, key: string, val: any) {
      target[key] = val
      ;(changeHooks[key] || []).forEach(fn => fn(val))
      return true
    },
  })
}
