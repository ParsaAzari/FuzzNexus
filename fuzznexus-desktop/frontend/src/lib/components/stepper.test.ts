import { describe, it, expect } from 'vitest';

describe('NumberStepper Logic & Boundary Handling', () => {
  function clamp(val: number, min: number, max: number): number {
    return Math.min(max, Math.max(min, val));
  }

  function stepValue(current: number, delta: number, min: number, max: number): number {
    return clamp(current + delta, min, max);
  }

  function parseInput(raw: string, min: number, max: number): number {
    let num = parseInt(raw, 10);
    if (isNaN(num)) num = min;
    return clamp(num, min, max);
  }

  it('increments threads within bounds (1 to 100)', () => {
    expect(stepValue(10, 1, 1, 100)).toBe(11);
    expect(stepValue(10, 5, 1, 100)).toBe(15); // shiftStep
    expect(stepValue(99, 5, 1, 100)).toBe(100); // capped at max
    expect(stepValue(100, 1, 1, 100)).toBe(100);
  });

  it('decrements threads within bounds (1 to 100)', () => {
    expect(stepValue(10, -1, 1, 100)).toBe(9);
    expect(stepValue(4, -5, 1, 100)).toBe(1); // floor at min
    expect(stepValue(1, -1, 1, 100)).toBe(1);
  });

  it('steps rate correctly (0 to 5000 with step=10, shiftStep=50)', () => {
    expect(stepValue(0, 10, 0, 5000)).toBe(10);
    expect(stepValue(50, 50, 0, 5000)).toBe(100);
    expect(stepValue(4980, 50, 0, 5000)).toBe(5000);
    expect(stepValue(0, -10, 0, 5000)).toBe(0);
  });

  it('handles invalid or non-numeric inputs gracefully', () => {
    expect(parseInput('', 1, 100)).toBe(1);
    expect(parseInput('abc', 1, 100)).toBe(1);
    expect(parseInput('-50', 1, 100)).toBe(1);
    expect(parseInput('999', 1, 100)).toBe(100);
    expect(parseInput('42', 1, 100)).toBe(42);
  });
});
