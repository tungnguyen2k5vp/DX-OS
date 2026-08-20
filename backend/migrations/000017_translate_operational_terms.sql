-- Replace technical terms in existing user-facing AI recommendations.
UPDATE ai_recommendations
SET title = 'Phiếu quá hạn cần ưu tiên xử lý',
    summary = 'Phiếu đang chờ xử lý đã quá thời hạn xử lý. Kiểm tra người phụ trách và nguyên nhân tắc nghẽn.'
WHERE recommendation_type = 'SLA_BREACH_RISK'
  AND title = 'Phiếu quá hạn SLA cần ưu tiên';
