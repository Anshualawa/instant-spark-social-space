import { useEffect, useRef, useState } from 'react';

/**
 * A hook that provides a debounced version of a value or function.
 * 
 * @param value - The value or function to debounce
 * @param delay - The delay in milliseconds
 * @returns - The debounced value or function
 */
export function useDebounce<T>(value: T, delay: number): T;
export function useDebounce<T extends (...args: any[]) => any>(
  func: T,
  delay: number
): (...args: Parameters<T>) => void;

export function useDebounce<T>(value: T, delay: number): T | ((...args: any[]) => void) {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    // If value is a function, return a debounced function
    if (typeof value === 'function') {
      return;
    }

    // Otherwise, debounce the value
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }

    timeoutRef.current = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, [value, delay]);

  // If value is a function, return a debounced version of the function
  if (typeof value === 'function') {
    return (...args: any[]) => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
      
      timeoutRef.current = setTimeout(() => {
        (value as Function)(...args);
      }, delay);
    };
  }

  // Otherwise, return the debounced value
  return debouncedValue;
}
