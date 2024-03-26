resource "aws_s3_bucket" "ztmf_web_assets" {
  bucket = local.domain_name
}

resource "aws_s3_bucket_policy" "ztmf_web_assets_access_from_vpc" {
  bucket = aws_s3_bucket.ztmf_web_assets.id
  policy = data.aws_iam_policy_document.allow_s3_access_from_vpc_endpoint.json
}

data "aws_iam_policy_document" "allow_s3_access_from_vpc_endpoint" {
  statement {
    principals {
      type        = "*"
      identifiers = ["*"]
    }

    actions = [
      "s3:GetObject",
      "s3:ListBucket",
    ]

    resources = [
      aws_s3_bucket.ztmf_web_assets.arn,
      "${aws_s3_bucket.ztmf_web_assets.arn}/*",
    ]

    condition {
      test     = "StringEquals"
      variable = "aws:SourceVpce"
      values   = [aws_vpc_endpoint.ztmf["s3"].id]
    }
  }

  statement {
    sid    = "AllowSSLRequestsOnly"
    effect = "Deny"

    principals {
      type        = "*"
      identifiers = ["*"]
    }

    actions = [
      "s3:*"
    ]

    resources = [
      aws_s3_bucket.ztmf_web_assets.arn,
      "${aws_s3_bucket.ztmf_web_assets.arn}/*",
    ]


    condition {
      test     = "Bool"
      variable = "aws:SecureTransport"
      values   = ["false"]
    }
  }
}
