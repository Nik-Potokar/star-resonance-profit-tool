import React, { useState, useEffect, useRef } from 'react';
import './App.css';
import { SetGoodPrice, GetAllBestProfit } from "../wailsjs/go/main/App";
import { Debug } from '../wailsjs/go/main/App';
import { SavePrice, RunCrawl} from '../wailsjs/go/main/App';

export default function App() {
    // State management
    const [profits, setProfits] = useState([]);
    const [loading, setLoading] = useState(true);
    const [editingPrice, setEditingPrice] = useState(null); // Currently editing price item
    const priceInputRef = useRef(null);
    const [errorMessage, setErrorMessage] = useState(''); // Error message
    const [showConfirmDialog, setShowConfirmDialog] = useState(false); // Confirmation dialog display status

    // Search-related state
    const [searchTerm, setSearchTerm] = useState('');
    const [matchedIndices, setMatchedIndices] = useState([]);
    const [currentMatchIndex, setCurrentMatchIndex] = useState(0);
    const goodsRowRefs = useRef([]);

    // Load profit data
    useEffect(() => {
        async function fetchProfits() {
            try {
                const data = await GetAllBestProfit();
                setProfits(data);
                setLoading(false);
            } catch (error) {
                console.error("Failed to get profit data:", error);
                setLoading(false);
            }
        }
        
        fetchProfits();
    }, []);

    // Handle price double-click edit
    const handlePriceDoubleClick = (profit, index) => {
        setEditingPrice({
            index,
            name: profit.name,
            value: profit.price || 0
        });
        // Use setTimeout to ensure DOM element is rendered before focusing
        setTimeout(() => {
            if (priceInputRef.current) {
                priceInputRef.current.focus();
                priceInputRef.current.select();
            }
        }, 10);
    };

    // Handle price edit completion
    const handlePriceEditComplete = async (save = true) => {
        if (save && editingPrice) {
            try {
                const newPrice = parseInt(editingPrice.value, 10);
                if (!isNaN(newPrice) && newPrice >= 0) {
                    // Call backend API to update price
                    await SetGoodPrice(editingPrice.name, newPrice);

                    // Update local state
                    setProfits(prevProfits => {
                        const newProfits = [...prevProfits];
                        newProfits[editingPrice.index].price = newPrice;
                        return newProfits;
                    });
                    const data = await GetAllBestProfit();
                    setProfits(data);
                    setLoading(false);
                    SavePrice()
                }
            } catch (error) {
                console.error("Failed to update price:", error);
            }
        }
        setEditingPrice(null);
    };

    // Handle price input change
    const handlePriceChange = (e) => {
        setEditingPrice(prev => ({
            ...prev,
            value: e.target.value
        }));
    };

    // Handle keyboard events
    const handleKeyDown = (e) => {
        if (e.key === 'Enter') {
            handlePriceEditComplete(true);
        } else if (e.key === 'Escape') {
            handlePriceEditComplete(false);
        }
    };

    // Format number to string with thousand separators
    const formatNumber = (num) => {
        if (!num) return "0";

        // Handle decimals and integers
        const parts = num.toString().split('.');
        parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ",");

        // If there's a decimal part, add it back
        return parts.length > 1 ? parts[0] : parts[0];
    };

    // Handle crawl data button click
    const handleCrawlClick = () => {
        // Show confirmation dialog
        setShowConfirmDialog(true);
    };

    // Confirm crawl data
    const confirmCrawl = async () => {
        // Hide confirmation dialog
        setShowConfirmDialog(false);

        try {
            // Clear previous error message
            setErrorMessage('');

            // Show loading state
            setLoading(true);

            // Call crawl data method
            await RunCrawl();

            // Reload data
            const data = await GetAllBestProfit();
            setProfits(data);
        } catch (error) {
            console.error("Failed to crawl data:", error);
            // Set error message
            setErrorMessage(error.toString());
        } finally {
            setLoading(false);
        }
    };
    
    // Cancel crawl data
    const cancelCrawl = () => {
        // Hide confirmation dialog
        setShowConfirmDialog(false);
    };

    // Handle search input change
    const handleSearchChange = (e) => {
        const value = e.target.value;
        setSearchTerm(value);

        if (value.trim() === '') {
            setMatchedIndices([]);
            setCurrentMatchIndex(0);
            return;
        }

        // Find matches
        const matches = profits.reduce((acc, profit, index) => {
            if (profit.name.toLowerCase().includes(value.toLowerCase())) {
                acc.push(index);
            }
            return acc;
        }, []);
        
        setMatchedIndices(matches);
        setCurrentMatchIndex(matches.length > 0 ? 0 : -1);

        // If there are matches, scroll to the first match
        if (matches.length > 0) {
            setTimeout(() => {
                scrollToMatch(matches[0]);
            }, 100);
        }
    };
    
    // Handle search box keyboard events
    const handleSearchKeyDown = (e) => {
        // When Enter is pressed, jump to next match
        if (e.key === 'Enter') {
            e.preventDefault();
            goToNextMatch();
        }
    };
    
    // Jump to next match
    const goToNextMatch = () => {
        if (matchedIndices.length === 0) return;
        
        const nextIndex = (currentMatchIndex + 1) % matchedIndices.length;
        setCurrentMatchIndex(nextIndex);
        scrollToMatch(matchedIndices[nextIndex]);
    };
    
    // Scroll to specified match
    const scrollToMatch = (rowIndex) => {
        if (goodsRowRefs.current[rowIndex]) {
            goodsRowRefs.current[rowIndex].scrollIntoView({
                behavior: 'smooth',
                block: 'center'
            });
        }
    };

    return (
        <div className="app-container">
            {/* Confirmation dialog */}
            {showConfirmDialog && (
                <div className="confirm-dialog-overlay">
                    <div className="confirm-dialog">
                        <h3>Please confirm the following before crawling:</h3>
                        <ol>
                            <li>Game must be in windowed mode at 1080p resolution.</li>
                            <li>Interface should be at the trading center, with the left sidebar not manually collapsed or moved.</li>
                            <li>Run this tool with administrator privileges. After confirmation, it will automatically switch to the game interface to start collecting.</li>
                            <li>Collection will complete in approximately 2-3 minutes after reaching [Weaving]. There may be data errors that can be manually corrected.</li>
                        </ol>
                        <div className="confirm-dialog-buttons">
                            <button
                                className="confirm-btn"
                                onClick={confirmCrawl}
                            >
                                Confirm
                            </button>
                            <button
                                className="cancel-btn"
                                onClick={cancelCrawl}
                            >
                                Cancel
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {/* Floating error message display */}
            {errorMessage && (
                <div className="floating-error-container">
                    <div className="error-message">
                        <span>{errorMessage}</span>
                        <button
                            className="close-error-btn"
                            onClick={() => setErrorMessage('')}
                            title="Close error message"
                        >
                            ×
                        </button>
                    </div>
                </div>
            )}

            <div className="content-container">
                {/* Top toolbar */}
                <div className="toolbar-container">
                    {/* Crawl data button */}
                    <button
                        className="crawl-btn"
                        onClick={handleCrawlClick}
                        title="Crawl latest data"
                        disabled={loading}
                    >
                        {loading ? "Crawling..." : "Crawl Data"}
                    </button>

                    {/* Search box */}
                    <div className="search-container">
                        <input
                            type="text"
                            className="search-input"
                            placeholder="Search item name... (Press Enter to navigate)"
                            value={searchTerm}
                            onChange={handleSearchChange}
                            onKeyDown={handleSearchKeyDown}
                        />
                        {matchedIndices.length > 0 && (
                            <div className="search-results">
                                <span className="match-count">
                                    {currentMatchIndex + 1}/{matchedIndices.length}
                                </span>
                                <button
                                    className="next-match-btn"
                                    onClick={goToNextMatch}
                                    title="Find next match"
                                >
                                    ↓
                                </button>
                            </div>
                        )}
                    </div>
                </div>

                <div className="goods-list">
                    {loading ? (
                        <div className="loading">Loading...</div>
                    ) : (
                        <div className="goods-table">
                            <div className="goods-header">
                                <div className="goods-cell">Name</div>
                                <div className="goods-cell">Reference Price</div>
                                <div className="goods-cell profit-cell">Profit</div>
                                <div className="goods-cell comment-cell header-comment-cell">Details</div>
                            </div>
                            {profits.map((profit, index) => (
                                <div
                                    key={index}
                                    className={`goods-row ${matchedIndices.includes(index) ? 'matched-row' : ''} ${matchedIndices[currentMatchIndex] === index ? 'current-match' : ''}`}
                                    ref={el => goodsRowRefs.current[index] = el}
                                >
                                    <div className="goods-cell">{profit.name}</div>
                                    <div
                                        className="goods-cell price-cell"
                                        onDoubleClick={() => handlePriceDoubleClick(profit, index)}
                                    >
                                        {editingPrice && editingPrice.index === index ? (
                                            <input
                                                ref={priceInputRef}
                                                type="number"
                                                className="price-input"
                                                value={editingPrice.value}
                                                onChange={handlePriceChange}
                                                onBlur={() => handlePriceEditComplete(true)}
                                                onKeyDown={handleKeyDown}
                                                min="0"
                                            />
                                        ) : (
                                            <div className="price-display">
                                                {formatNumber(profit.price)}
                                                <span className="edit-hint">Double-click to edit</span>
                                            </div>
                                        )}
                                    </div>
                                    <div className="goods-cell profit-cell">
                                        <span className={profit.num > 0 ? "profit-positive" : "profit-negative"}>
                                            {formatNumber(profit.num)}
                                        </span>
                                    </div>
                                    <div className="goods-cell comment-cell">
                                        <div className="comment-content">
                                            <span className="comment-preview">
                                                {profit.comment ? "..." : "-"}
                                            </span>
                                            {profit.comment && (
                                                <div className="comment-tooltip">
                                                    {profit.comment.split('\n').map((line, i) => (
                                                        <React.Fragment key={i}>
                                                            {line}
                                                            {i < profit.comment.split('\n').length - 1 && <br />}
                                                        </React.Fragment>
                                                    ))}
                                                </div>
                                            )}
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}