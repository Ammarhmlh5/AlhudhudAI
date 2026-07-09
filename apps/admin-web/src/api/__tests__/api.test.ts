import { describe, it, expect, vi, beforeEach } from 'vitest'
import axios from 'axios'
import * as api from '../index'

vi.mock('../config', () => ({
  API_BASE_URL: 'http://localhost:8081',
}))

vi.mock('axios')

describe('API Module', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('getTenantStats', () => {
    it('returns tenant statistics successfully', async () => {
      const mockStats = {
        data: {
          totalTenants: 150,
          activeTenants: 120,
          messagesToday: 50000,
          revenue: 25000.50
        }
      }
      
      vi.mocked(axios.get).mockResolvedValue(mockStats)
      
      const result = await api.getTenantStats()
      
      expect(axios.get).toHaveBeenCalledWith('/api/v1/admin/stats/tenants')
      expect(result).toEqual(mockStats.data)
    })

    it('throws error when API fails', async () => {
      const error = new Error('Network error')
      vi.mocked(axios.get).mockRejectedValue(error)
      
      await expect(api.getTenantStats()).rejects.toThrow('Network error')
    })
  })

  describe('sendMessage', () => {
    it('sends message successfully', async () => {
      const request = {
        to: '+966501234567',
        content: 'Test message',
        channel: 'direct'
      }
      
      const mockResponse = {
        data: {
          message_id: 'msg-123',
          status: 'queued'
        }
      }
      
      vi.mocked(axios.post).mockResolvedValue(mockResponse)
      
      const result = await api.sendMessage(request)
      
      expect(axios.post).toHaveBeenCalledWith('/api/v1/messages/send', request)
      expect(result.message_id).toBe('msg-123')
    })
  })
})