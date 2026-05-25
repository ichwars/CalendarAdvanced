import { useEffect } from 'react';

const scrollKeys = new Set(['ArrowDown', 'ArrowLeft', 'ArrowRight', 'ArrowUp', 'End', 'Home', 'PageDown', 'PageUp', ' ']);

export function ScrollbarController() {
  useEffect(() => {
    const timeouts = new WeakMap<HTMLElement, number>();
    const activeElements = new Set<HTMLElement>();
    let hoveredScrollable: HTMLElement | null = null;

    function showScrollbar(element: HTMLElement) {
      element.dataset.scrollbarActive = 'true';
      activeElements.add(element);
      window.clearTimeout(timeouts.get(element));
      timeouts.set(element, window.setTimeout(() => {
        delete element.dataset.scrollbarActive;
        activeElements.delete(element);
      }, 900));
    }

    function handleScroll(event: Event) {
      showScrollbar(resolveScrollElement(event.target));
    }

    function handleWheel(event: WheelEvent) {
      const target = findScrollableParent(event.target);
      showScrollbar(target === document.documentElement && hoveredScrollable ? hoveredScrollable : target);
    }

    function handleKeydown(event: KeyboardEvent) {
      if (scrollKeys.has(event.key)) {
        showScrollbar(findScrollableParent(document.activeElement));
      }
    }

    function handlePointerMove(event: PointerEvent) {
      const target = findScrollableParent(event.target);
      hoveredScrollable = target === document.documentElement ? null : target;
    }

    window.addEventListener('scroll', handleScroll, true);
    window.addEventListener('wheel', handleWheel, { passive: true });
    window.addEventListener('keydown', handleKeydown);
    window.addEventListener('pointermove', handlePointerMove, { passive: true });
    return () => {
      activeElements.forEach((element) => {
        window.clearTimeout(timeouts.get(element));
        delete element.dataset.scrollbarActive;
      });
      window.removeEventListener('scroll', handleScroll, true);
      window.removeEventListener('wheel', handleWheel);
      window.removeEventListener('keydown', handleKeydown);
      window.removeEventListener('pointermove', handlePointerMove);
    };
  }, []);

  return null;
}

function resolveScrollElement(target: EventTarget | null): HTMLElement {
  if (target === document || target === window || target === document.body || target === document.documentElement) {
    return document.documentElement;
  }
  return target instanceof HTMLElement ? target : document.documentElement;
}

function findScrollableParent(target: EventTarget | null): HTMLElement {
  let element = target instanceof HTMLElement ? target : document.activeElement;
  while (element instanceof HTMLElement && element !== document.body) {
    if (isScrollable(element)) {
      return element;
    }
    element = element.parentElement;
  }
  return document.documentElement;
}

function isScrollable(element: HTMLElement): boolean {
  const style = window.getComputedStyle(element);
  const canScrollY = element.scrollHeight > element.clientHeight && /(auto|scroll|overlay)/.test(style.overflowY);
  const canScrollX = element.scrollWidth > element.clientWidth && /(auto|scroll|overlay)/.test(style.overflowX);
  return canScrollY || canScrollX;
}
