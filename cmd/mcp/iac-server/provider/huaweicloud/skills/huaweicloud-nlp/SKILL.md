---
name: huaweicloud-nlp
description: HuaweiCloud NLP API guide. 27 APIs covering 机器翻译服务, 自然语言处理基础服务, 语言理解服务, 语言生成服务.
---

# HuaweiCloud NLP API Guide

27 APIs. Tags: 机器翻译服务, 自然语言处理基础服务, 语言理解服务, 语言生成服务

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `RunAspectSentiment` | POST | `/v1/{project_id}/nlu/aspect-sentiment` | 属性级情感分析 |
| `RunAspectSentimentAdvance` | POST | `/v1/{project_id}/nlu/aspect-sentiment/advance` | 属性级情感分析(高级版) |
| `RunClassification` | POST | `/v1/{project_id}/nlu/classification` | 文本分类 |
| `RunConstituencyParser` | POST | `/v1/{project_id}/nlp-fundamental/constituency-parser` | 成分句法分析 |
| `RunDependencyParser` | POST | `/v1/{project_id}/nlp-fundamental/dependency-parser` | 依存句法分析 |
| `RunDocClassification` | POST | `/v1/{project_id}/nlu/doc-classification` | 文档分类 |
| `RunDomainSentiment` | POST | `/v1/{project_id}/nlu/sentiment/domain` | 情感分析(领域版) |
| `RunEntityLinking` | POST | `/v1/{project_id}/nlp-fundamental/entity-linking` | 实体链接 |
| `RunEntitySentiment` | POST | `/v1/{project_id}/nlu/entity-sentiment` | 实体级情感分析 |
| `RunEventExtraction` | POST | `/v1/{project_id}/nlp-fundamental/event-extraction` | 事件抽取 |
| `RunFileTranslation` | POST | `/v1/{project_id}/machine-translation/file-translation/jobs` | 文档翻译 |
| `RunGetFileTranslationResult` | GET | `/v1/{project_id}/machine-translation/file-translation/jobs/{job_id}` | 文档翻译状态查询 |
| `RunKeywordExtract` | POST | `/v1/{project_id}/nlp-fundamental/keyword-extraction` | 关键词抽取 |
| `RunLanguageDetection` | POST | `/v1/{project_id}/machine-translation/language-detection` | 语种识别 |
| `RunMultiGrainedSegment` | POST | `/v1/{project_id}/nlp-fundamental/multi-grained-segment` | 多粒度分词 |
| `RunNer` | POST | `/v1/{project_id}/nlp-fundamental/ner` | 命名实体识别(基础版) |
| `RunNerDomain` | POST | `/v1/{project_id}/nlp-fundamental/ner/domain` | 命名实体识别(领域版) |
| `RunPoem` | POST | `/v1/{project_id}/nlg/poem` | 诗歌生成 |
| `RunSegment` | POST | `/v1/{project_id}/nlp-fundamental/segment` | 分词 |
| `RunSemanticParser` | POST | `/v1/{project_id}/nlu/semantic-parser` | 意图理解 |
| `RunSentenceEmbedding` | POST | `/v1/{project_id}/nlp-fundamental/sentence-embedding` | 句向量 |
| `RunSentiment` | POST | `/v1/{project_id}/nlu/sentiment` | 情感分析(基础版) |
| `RunSummary` | POST | `/v1/{project_id}/nlg/summarization` | 文本摘要(基础版) |
| `RunSummaryDomain` | POST | `/v1/{project_id}/nlg/summarization/domain` | 文本摘要(领域版) |
| `RunTextSimilarity` | POST | `/v1/{project_id}/nlp-fundamental/text-similarity` | 文本相似度(基础版) |
| `RunTextSimilarityAdvance` | POST | `/v1/{project_id}/nlp-fundamental/text-similarity/advance` | 文本相似度(高级版) |
| `RunTextTranslation` | POST | `/v1/{project_id}/machine-translation/text-translation` | 文本翻译 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
