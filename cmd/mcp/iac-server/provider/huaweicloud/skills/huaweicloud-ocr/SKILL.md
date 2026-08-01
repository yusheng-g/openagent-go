---
name: huaweicloud-ocr
description: HuaweiCloud OCR API guide. 38 APIs covering VIN码识别, 不动产证识别, 保险单识别, 出租车发票识别, 印章识别.
---

# HuaweiCloud OCR API Guide

38 APIs. Tags: VIN码识别, 不动产证识别, 保险单识别, 出租车发票识别, 印章识别, 发票验真, 名片识别, 哥伦比亚身份证识别, 增值税发票识别, 定额发票识别, 户口本识别, 手写文字识别, 承兑汇票识别, 护照识别, 智能分类识别, 智能文档解析, 机动车销售发票识别, 泰国车牌识别, 泰文身份证识别, 火车票识别, 电子面单识别, 网络图片识别, 自定义模板OCR, 营业执照识别, 行驶证识别, 财务报表识别, 身份证识别, 车牌识别, 车辆合格证识别, 车辆通行费发票识别, 通用文字识别, 通用表格识别, 道路运输从业资格证识别, 道路运输证识别, 银行卡识别, 银行回单识别, 飞机行程单识别, 驾驶证识别

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `RecognizeAcceptanceBill` | POST | `/v2/{project_id}/ocr/acceptance-bill` | 承兑汇票识别 |
| `RecognizeAutoClassification` | POST | `/v2/{project_id}/ocr/auto-classification` | 智能分类识别 |
| `RecognizeBankcard` | POST | `/v2/{project_id}/ocr/bankcard` | 银行卡识别 |
| `RecognizeBankReceipt` | POST | `/v2/{project_id}/ocr/bank-receipt` | 银行回单识别 |
| `RecognizeBusinessCard` | POST | `/v2/{project_id}/ocr/business-card` | 名片识别 |
| `RecognizeBusinessLicense` | POST | `/v2/{project_id}/ocr/business-license` | 营业执照识别 |
| `RecognizeColombiaIdCard` | POST | `/v2/{project_id}/ocr/colombia-id-card` | 哥伦比亚身份证识别 |
| `RecognizeCustomTemplate` | POST | `/v2/{project_id}/ocr/custom-template` | 自定义模板OCR |
| `RecognizeDriverLicense` | POST | `/v2/{project_id}/ocr/driver-license` | 驾驶证识别 |
| `RecognizeFinancialStatement` | POST | `/v2/{project_id}/ocr/financial-statement` | 财务报表识别 |
| `RecognizeFlightItinerary` | POST | `/v2/{project_id}/ocr/flight-itinerary` | 飞机行程单识别 |
| `RecognizeGeneralTable` | POST | `/v2/{project_id}/ocr/general-table` | 通用表格识别 |
| `RecognizeGeneralText` | POST | `/v2/{project_id}/ocr/general-text` | 通用文字识别 |
| `RecognizeHandwriting` | POST | `/v2/{project_id}/ocr/handwriting` | 手写文字识别 |
| `RecognizeHouseholdRegister` | POST | `/v2/{project_id}/ocr/household-register` | 户口本识别 |
| `RecognizeIdCard` | POST | `/v2/{project_id}/ocr/id-card` | 身份证识别 |
| `RecognizeInsurancePolicy` | POST | `/v2/{project_id}/ocr/insurance-policy` | 保险单识别 |
| `RecognizeInvoiceVerification` | POST | `/v2/{project_id}/ocr/invoice-verification` | 发票验真 |
| `RecognizeLicensePlate` | POST | `/v2/{project_id}/ocr/license-plate` | 车牌识别 |
| `RecognizeMvsInvoice` | POST | `/v2/{project_id}/ocr/mvs-invoice` | 机动车销售发票识别 |
| `RecognizePassport` | POST | `/v2/{project_id}/ocr/passport` | 护照识别 |
| `RecognizeQualificationCertificate` | POST | `/v2/{project_id}/ocr/transportation-qualification-certificate` | 道路运输从业资格证识别 |
| `RecognizeQuotaInvoice` | POST | `/v2/{project_id}/ocr/quota-invoice` | 定额发票识别 |
| `RecognizeRealEstateCertificate` | POST | `/v2/{project_id}/ocr/real-estate-certificate` | 不动产证识别 |
| `RecognizeSeal` | POST | `/v2/{project_id}/ocr/seal` | 印章识别 |
| `RecognizeSmartDocumentRecognizer` | POST | `/v2/{project_id}/ocr/smart-document-recognizer` | 智能文档解析 |
| `RecognizeTaxiInvoice` | POST | `/v2/{project_id}/ocr/taxi-invoice` | 出租车发票识别 |
| `RecognizeThailandIdcard` | POST | `/v2/{project_id}/ocr/thailand-id-card` | 泰文身份证识别 |
| `RecognizeThailandLicensePlate` | POST | `/v2/{project_id}/ocr/thailand-license-plate` | 泰国车牌识别 |
| `RecognizeTollInvoice` | POST | `/v2/{project_id}/ocr/toll-invoice` | 车辆通行费发票识别 |

... and 8 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
