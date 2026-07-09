import apiClient from './axios';
const API_URL = ''; // Now handled by apiClient baseURL

interface MessageRecipient {
  phone_number: string;
  message: string;
}

interface SendMessageData {
  recipients: MessageRecipient[];
  scheduled_at?: string;
}

interface SendMessageResponse {
  batch_id: string;
  total_messages: number;
  estimated_cost: number;
  currency: string;
  status: string;
}

interface CostEstimate {
  total_segments: number;
  cost_per_segment: number;
  total_cost: number;
  currency: string;
  encoding: string;
  breakdown: {
    phone_number: string;
    segments: number;
    cost: number;
  }[];
}

interface CalculateCostData {
  recipients: MessageRecipient[];
}

/**
 * حساب تكلفة الرسائل قبل الإرسال
 * @param data بيانات الرسائل المراد حساب تكلفتها
 * @returns تقدير التكلفة الإجمالية وتفاصيل كل رسالة
 */
export const calculateCost = async (data: CalculateCostData): Promise<CostEstimate | null> => {
  try {
    const response = await apiClient.post(`${API_URL}/api/messages/calculate-cost`, data);

    return response.data;
  } catch (error: any) {
    console.error('Error calculating cost:', error);
    throw new Error(error.response?.data?.error || 'فشل في حساب التكلفة');
  }
};

/**
 * إرسال رسائل مع حساب التكلفة تلقائياً
 * @param data بيانات الرسائل المراد إرسالها
 * @returns معلومات الدفعة والتكلفة المقدرة
 */
export const sendMessage = async (data: SendMessageData): Promise<SendMessageResponse | null> => {
  try {
    // حساب التكلفة أولاً
    const costEstimate = await calculateCost({ recipients: data.recipients });
    
    if (!costEstimate) {
      throw new Error('فشل في حساب التكلفة');
    }

    // إرسال الرسائل مع التكلفة المقدرة
    const response = await apiClient.post(
      `${API_URL}/api/messages/send`,
      {
        ...data,
        estimated_cost: costEstimate.total_cost,
        total_segments: costEstimate.total_segments,
      }
    );

    return response.data;
  } catch (error: any) {
    console.error('Error sending message:', error);
    throw new Error(error.response?.data?.error || 'فشل في إرسال الرسالة');
  }
};

/**
 * حساب عدد أجزاء الرسالة بناءً على المحتوى
 * @param message نص الرسالة
 * @returns عدد الأجزاء ونوع الترميز
 */
export const calculateSegments = (message: string): { segments: number; encoding: string } => {
  // التحقق من نوع الترميز
  const isGSM7 = /^[@£$¥èéùìòÇØøÅåΔ_ΦΓΛΩΠΨΣΘΞÆæßÉ !"#¤%&'()*+,\-./0-9:;<=>?¡A-ZÄÖÑÜ§¿a-zäöñüà\r\n\^{}\\[~\]|€]*$/.test(
    message
  );

  const encoding = isGSM7 ? 'GSM-7' : 'Unicode';
  const length = message.length;

  let segments: number;

  if (isGSM7) {
    // GSM-7: الجزء الأول 160 حرف، الأجزاء التالية 153 حرف
    if (length <= 160) {
      segments = 1;
    } else {
      segments = Math.ceil(length / 153);
    }
  } else {
    // Unicode: الجزء الأول 70 حرف، الأجزاء التالية 67 حرف
    if (length <= 70) {
      segments = 1;
    } else {
      segments = Math.ceil(length / 67);
    }
  }

  return { segments, encoding };
};

/**
 * حساب التكلفة المحلية (بدون استدعاء API)
 * ملاحظة: هذه دالة تقريبية للعرض السريع فقط
 * @param message نص الرسالة
 * @param costPerSegment تكلفة الجزء الواحد
 * @returns التكلفة المقدرة
 */
export const estimateCostLocal = (message: string, costPerSegment: number = 0.05): number => {
  const { segments } = calculateSegments(message);
  return segments * costPerSegment;
};

/**
 * جلب سجل الرسائل المرسلة
 * @param filters فلاتر البحث (اختياري)
 * @returns قائمة الرسائل
 */
export const getMessageHistory = async (filters?: {
  from_date?: string;
  to_date?: string;
  status?: string;
  limit?: number;
  offset?: number;
}) => {
  try {
    const params = new URLSearchParams();
    if (filters?.from_date) params.append('from_date', filters.from_date);
    if (filters?.to_date) params.append('to_date', filters.to_date);
    if (filters?.status) params.append('status', filters.status);
    if (filters?.limit) params.append('limit', String(filters.limit));
    if (filters?.offset) params.append('offset', String(filters.offset));

    const response = await apiClient.get(`${API_URL}/api/messages/history?${params.toString()}`);

    return response.data;
  } catch (error: any) {
    console.error('Error fetching message history:', error);
    throw new Error(error.response?.data?.error || 'فشل في جلب سجل الرسائل');
  }
};

/**
 * جلب تفاصيل دفعة رسائل
 * @param batchId معرف الدفعة
 * @returns تفاصيل الدفعة
 */
export const getBatchDetails = async (batchId: string) => {
  try {
    const response = await apiClient.get(`${API_URL}/api/messages/batch/${batchId}`);

    return response.data;
  } catch (error: any) {
    console.error('Error fetching batch details:', error);
    throw new Error(error.response?.data?.error || 'فشل في جلب تفاصيل الدفعة');
  }
};

/**
 * إلغاء دفعة رسائل مجدولة
 * @param batchId معرف الدفعة
 * @returns نتيجة الإلغاء
 */
export const cancelScheduledBatch = async (batchId: string) => {
  try {
    const response = await apiClient.post(`${API_URL}/api/messages/batch/${batchId}/cancel`, {});

    return response.data;
  } catch (error: any) {
    console.error('Error canceling batch:', error);
    throw new Error(error.response?.data?.error || 'فشل في إلغاء الدفعة');
  }
};
