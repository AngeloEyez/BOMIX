import { clsx } from "clsx";
import { twMerge } from "tailwind-merge";

/**
 * 合併 tailwind classes，解決樣式衝突與條件渲染問題。
 *
 * @param {...(string|Object|Array)} inputs - clsx 接受的 class 參數
 * @returns {string} 合併後的 tailwind class string
 */
export function cn(...inputs) {
  return twMerge(clsx(inputs));
}
