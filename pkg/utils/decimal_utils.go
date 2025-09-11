package utils

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ConvertFloatToDecimal128 converts float64 to MongoDB Decimal128
func ConvertFloatToDecimal128(value float64) (primitive.Decimal128, error) {
	// Convert float to string with proper precision
	str := fmt.Sprintf("%.8f", value)
	return primitive.ParseDecimal128(str)
}

// ConvertDecimal128ToFloat converts MongoDB Decimal128 to float64
func ConvertDecimal128ToFloat(decimal primitive.Decimal128) (float64, error) {
	str := decimal.String()
	return strconv.ParseFloat(str, 64)
}

// SumDecimal128 adds two Decimal128 values
func SumDecimal128(a, b primitive.Decimal128) (primitive.Decimal128, error) {
	aFloat, err := ConvertDecimal128ToFloat(a)
	if err != nil {
		return primitive.Decimal128{}, err
	}

	bFloat, err := ConvertDecimal128ToFloat(b)
	if err != nil {
		return primitive.Decimal128{}, err
	}

	sum := aFloat + bFloat
	return ConvertFloatToDecimal128(sum)
}

// SubtractDecimal128 subtracts b from a
func SubtractDecimal128(a, b primitive.Decimal128) (primitive.Decimal128, error) {
	aFloat, err := ConvertDecimal128ToFloat(a)
	if err != nil {
		return primitive.Decimal128{}, err
	}

	bFloat, err := ConvertDecimal128ToFloat(b)
	if err != nil {
		return primitive.Decimal128{}, err
	}

	result := aFloat - bFloat
	return ConvertFloatToDecimal128(result)
}

// CompareDecimal128 compares two Decimal128 values for equality
func CompareDecimal128(a, b primitive.Decimal128) (bool, error) {
	aFloat, err := ConvertDecimal128ToFloat(a)
	if err != nil {
		return false, err
	}

	bFloat, err := ConvertDecimal128ToFloat(b)
	if err != nil {
		return false, err
	}

	// Use epsilon for floating point comparison
	epsilon := 1e-8
	return math.Abs(aFloat-bFloat) < epsilon, nil
}

// IsDecimal128Zero checks if a Decimal128 value is zero
func IsDecimal128Zero(decimal primitive.Decimal128) (bool, error) {
	zeroDecimal, err := ConvertFloatToDecimal128(0.0)
	if err != nil {
		return false, err
	}

	return CompareDecimal128(decimal, zeroDecimal)
}

// IsDecimal128Negative checks if a Decimal128 value is negative
func IsDecimal128Negative(decimal primitive.Decimal128) (bool, error) {
	value, err := ConvertDecimal128ToFloat(decimal)
	if err != nil {
		return false, err
	}

	return value < 0, nil
}

// AbsDecimal128 returns the absolute value of a Decimal128
func AbsDecimal128(decimal primitive.Decimal128) (primitive.Decimal128, error) {
	value, err := ConvertDecimal128ToFloat(decimal)
	if err != nil {
		return primitive.Decimal128{}, err
	}

	return ConvertFloatToDecimal128(math.Abs(value))
}

// ValidateDecimal128Amount validates that a decimal amount is valid for financial operations
func ValidateDecimal128Amount(decimal primitive.Decimal128) error {
	value, err := ConvertDecimal128ToFloat(decimal)
	if err != nil {
		return err
	}

	if math.IsInf(value, 0) || math.IsNaN(value) {
		return errors.New("invalid decimal value: infinity or NaN")
	}

	// Check for reasonable financial limits (adjust as needed)
	if value > 1e15 || value < -1e15 {
		return errors.New("decimal value exceeds reasonable financial limits")
	}

	return nil
}
