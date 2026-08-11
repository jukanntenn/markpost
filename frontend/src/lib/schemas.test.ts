import { describe, expect, it } from 'vitest'
import {
  loginSchema,
  passwordChangeSchema,
  adminCreateUserSchema,
} from './schemas'

// B1.5/B1.7/B1.11/C2.3 zod schema 前后端规则对齐。
describe('loginSchema', () => {
  it('accepts any non-empty credentials (no length checks, B1.7)', () => {
    const r = loginSchema.safeParse({ username: 'a', password: 'b' })
    expect(r.success).toBe(true)
  })

  it('rejects empty username or password', () => {
    expect(loginSchema.safeParse({ username: '', password: 'x' }).success).toBe(
      false,
    )
    expect(loginSchema.safeParse({ username: 'x', password: '' }).success).toBe(
      false,
    )
  })
})

describe('passwordChangeSchema', () => {
  it('accepts passwords of length 8..72', () => {
    expect(
      passwordChangeSchema.safeParse({
        currentPassword: 'oldpass',
        newPassword: '12345678',
        confirmPassword: '12345678',
      }).success,
    ).toBe(true)
  })

  it('rejects passwords shorter than 8', () => {
    expect(
      passwordChangeSchema.safeParse({
        currentPassword: 'oldpass',
        newPassword: '1234567',
        confirmPassword: '1234567',
      }).success,
    ).toBe(false)
  })

  it('rejects passwords over 72 bytes (C2.3 双校验)', () => {
    const mixed = '界'.repeat(25) // 75 bytes
    expect(
      passwordChangeSchema.safeParse({
        currentPassword: 'oldpass',
        newPassword: mixed,
        confirmPassword: mixed,
      }).success,
    ).toBe(false)
  })

  it('rejects mismatched confirmation', () => {
    expect(
      passwordChangeSchema.safeParse({
        currentPassword: 'oldpass',
        newPassword: '12345678',
        confirmPassword: '87654321',
      }).success,
    ).toBe(false)
  })
})

describe('adminCreateUserSchema', () => {
  it('accepts valid user', () => {
    expect(
      adminCreateUserSchema.safeParse({
        username: 'bob',
        email: 'bob@example.com',
        password: 'password123',
      }).success,
    ).toBe(true)
  })

  it('allows empty email (OAuth-only user)', () => {
    expect(
      adminCreateUserSchema.safeParse({
        username: 'bob',
        email: '',
        password: 'password123',
      }).success,
    ).toBe(true)
  })

  it('rejects invalid email format', () => {
    expect(
      adminCreateUserSchema.safeParse({
        username: 'bob',
        email: 'not-an-email',
        password: 'password123',
      }).success,
    ).toBe(false)
  })
})
